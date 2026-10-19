package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type labCheckSocketKey struct{}

// Only the child process has access to the running lab and its interactive
// shell state. The dashboard asks that process to grade via a private socket.
var (
	errLabReflectionRequired = errors.New("reflection required before grading")
	errLabChildClosed = errors.New("le terminal enfant a été fermé ou sa session a été interrompue")
)

type labCheckRequest struct {
	Command    string   `json:"command"`
	Reflection []string `json:"reflection,omitempty"`
}

type labCheckResponse struct {
	Passed          bool   `json:"passed"`
	NeedsReflection bool   `json:"needs_reflection,omitempty"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
}

type labCheckBridge struct {
	listener net.Listener
	done     chan struct{}
}

func startLabCheckBridge(path string, check func(io.Writer, []string) (bool, error)) (*labCheckBridge, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("lab check socket path must be absolute")
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("create lab check socket: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, err
	}
	// Keep the readiness marker after Close so the parent can distinguish
	// a dead terminal from a terminal still starting.
	if err := os.WriteFile(path+".ready", nil, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("mark lab check bridge ready: %w", err)
	}
	bridge := &labCheckBridge{listener: listener, done: make(chan struct{})}
	go func() {
		defer close(bridge.done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(90 * time.Second))
				var request labCheckRequest
				if err := json.NewDecoder(io.LimitReader(conn, 4096)).Decode(&request); err != nil ||
					request.Command != "check" || len(request.Reflection) > 2 {
					_ = json.NewEncoder(conn).Encode(labCheckResponse{Error: "Requête de vérification invalide."})
					return
				}
				var output bytes.Buffer
				passed, err := check(&output, request.Reflection)
				response := labCheckResponse{Passed: passed, Output: output.String()}
				switch {
				case errors.Is(err, errLabReflectionRequired):
					response.NeedsReflection = true
				case err != nil:
					response.Error = err.Error()
				}
				_ = json.NewEncoder(conn).Encode(response)
			}(conn)
		}
	}()
	return bridge, nil
}

func (bridge *labCheckBridge) Close() {
	if bridge == nil {
		return
	}
	_ = bridge.listener.Close()
	<-bridge.done
	_ = os.Remove(bridge.listener.Addr().String())
}

func requestLabCheck(ctx context.Context, path string, reflection []string) (labCheckResponse, error) {
	if path == "" {
		return labCheckResponse{}, errors.New("aucun terminal enfant associé")
	}
	// Starting the child terminal and its sandbox is asynchronous.
	startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var conn net.Conn
	var err error
	for {
		conn, err = (&net.Dialer{}).DialContext(startCtx, "unix", path)
		if err == nil {
			break
		}
		if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
			if _, markerErr := os.Stat(path + ".ready"); markerErr == nil {
				return labCheckResponse{}, fmt.Errorf("%w (connexion : %v)", errLabChildClosed, err)
			}
		}
		if startCtx.Err() != nil {
			return labCheckResponse{}, fmt.Errorf("connexion au terminal enfant impossible (démarrage lent ou terminal fermé) : %w (dernière erreur : %v)", startCtx.Err(), err)
		}
		select {
		case <-startCtx.Done():
			return labCheckResponse{}, fmt.Errorf("le terminal enfant ne répond pas après son démarrage: %w", startCtx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(90 * time.Second))
	if err := json.NewEncoder(conn).Encode(labCheckRequest{Command: "check", Reflection: reflection}); err != nil {
		return labCheckResponse{}, err
	}
	var result labCheckResponse
	if err := json.NewDecoder(io.LimitReader(conn, 1<<20)).Decode(&result); err != nil {
		return labCheckResponse{}, fmt.Errorf("réponse du terminal enfant: %w", err)
	}
	return result, nil
}

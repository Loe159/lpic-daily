{
  description = "LPIC Daily — offline Linux learning";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in {
          default = pkgs.buildGoModule {
            pname = "lpic-daily";
            version = "0-unstable";
            src = self;
            vendorHash = "sha256-Rw5koA0SmaM+q7TA3wh3k3LCxKzHratpaADo36tbpwI=";
            subPackages = [ "cmd/lpic" ];
            checkPhase = ''
              runHook preCheck
              go test ./cmd/lpic ./internal/bootstrap ./internal/selfupdate
              runHook postCheck
            '';
            nativeBuildInputs = [ pkgs.makeWrapper ];
            postPatch = ''
              substituteInPlace packaging/systemd/lpic-daily-notify.service \
                --replace-fail '%h/.local/bin/lpic' "$out/bin/lpic"
              substituteInPlace packaging/desktop/lpic-daily.desktop \
                --replace-fail 'Exec=lpic tui' "Exec=$out/bin/lpic tui"
            '';
            postInstall = ''
              wrapProgram $out/bin/lpic \
                --set LPIC_DAILY_SKIP_SELF_INSTALL 1 \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.tmux pkgs.libnotify pkgs.python3 pkgs.openssh pkgs.libvirt pkgs.qemu ]}
              install -Dm644 packaging/desktop/lpic-daily.desktop $out/share/applications/lpic-daily.desktop
            '';
            meta = {
              description = "Daily offline LPIC-1 learning and isolated labs";
              homepage = "https://github.com/Loe159/lpic-daily";
              license = pkgs.lib.licenses.asl20;
              mainProgram = "lpic";
              platforms = systems;
            };
          };
        });
    };
}

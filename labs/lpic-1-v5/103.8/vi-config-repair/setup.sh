#!/usr/bin/env bash
set -euo pipefail
rm -rf /workspace/*
install -d -m 0777 /workspace /run/lpic
cat > /workspace/config.txt <<'EOF'
alpha=1
beta=old
remove=this
copy=shared
anchor=end
EOF
cat > /root/.bash_profile <<'EOF'
export EDITOR=nano
EOF
cat > /root/.vimrc <<'EOF'
set nocompatible
function! LpicLog(message)
  call writefile([a:message], '/run/lpic/vim-events', 'a')
endfunction
function! LpicOperator()
  call LpicLog('operator:' . v:event.operator)
endfunction
function! LpicCommand()
  let t = getcmdtype()
  if t ==# '/' || t ==# '?'
    call LpicLog('search')
  elseif t ==# ':'
    call LpicLog('command')
  endif
endfunction
augroup lpic_daily
  autocmd!
  autocmd InsertEnter * call LpicLog('insert')
  autocmd CursorMoved * call LpicLog('move')
  autocmd TextYankPost * call LpicOperator()
  autocmd CmdlineLeave * call LpicCommand()
  autocmd BufWritePost /workspace/config.txt call LpicLog('write')
augroup END
EOF
rm -f /run/lpic/vim-events /workspace/editor-tools.txt
chmod -R a+rwX /workspace /run/lpic

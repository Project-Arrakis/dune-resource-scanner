# Session-specific, not general-purpose: the PID and breakpoint address are
# hardcoded from one live run against the pre-storm baseline (seed 2,
# DeepDesert_1 pid 390735) and will need updating against any other process.
#
# Run bounded, never left armed indefinitely: launch via
#   sudo gdb -x trace-336.gdb -batch > out.log 2>&1 &
# then after a fixed wait, send SIGINT (not SIGKILL) to the gdb process
# specifically -- this interrupts the blocking `continue` cleanly, letting
# the script proceed to `delete breakpoints` / `detach` / `quit` on its own.
# A hard kill would skip that cleanup and leave an int3 byte patched into
# the live game server's executable memory, crashing it the next time that
# code executes. Always verify original bytes are restored after detach
# before considering the run safe.
set pagination off
set confirm off
attach 390735
break *0x56724571147B
commands 1
  printf "HIT rax=%#llx r14=%#llx rdi=%#llx rsi=%#llx rbx=%#llx\n", $rax, $r14, $rdi, $rsi, $rbx
  continue
end
continue
delete breakpoints
detach
quit

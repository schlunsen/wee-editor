#!/usr/bin/env python3
"""Give Rasterminal a PTY while its frames pass through the build UI compositor."""
import errno
import fcntl
import os
import pty
import select
import signal
import struct
import sys
import termios
import tty


def main():
    saved = termios.tcgetattr(0) if os.isatty(0) else None
    pid, master = pty.fork()
    if pid == 0:
        try:
            os.execvp(sys.argv[1], sys.argv[1:])
        except OSError:
            os._exit(127)

    def resize(*_):
        try:
            cols, rows = os.get_terminal_size(0)
            fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack('HHHH', rows, cols, 0, 0))
        except OSError:
            pass

    def stop(signum, _):
        try:
            os.kill(pid, signum)
        except ProcessLookupError:
            pass

    signal.signal(signal.SIGWINCH, resize)
    signal.signal(signal.SIGINT, stop)
    signal.signal(signal.SIGTERM, stop)
    resize()
    try:
        if saved:
            tty.setcbreak(0)
        inputs = [master, 0]
        while True:
            try:
                readable, _, _ = select.select(inputs, [], [])
                if master in readable:
                    data = os.read(master, 65536)
                    if not data:
                        break
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()
                if 0 in readable:
                    data = os.read(0, 1024)
                    if data:
                        os.write(master, data)
                    else:
                        inputs.remove(0)
            except OSError as error:
                if error.errno == errno.EIO:
                    break
                if error.errno != errno.EINTR:
                    raise
    finally:
        if saved:
            termios.tcsetattr(0, termios.TCSANOW, saved)
        os.close(master)
        # Also reap the renderer if the pipe was closed unexpectedly.
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        _, status = os.waitpid(pid, 0)
    code = os.waitstatus_to_exitcode(status)
    return 128 - code if code < 0 else code


if __name__ == '__main__':
    sys.exit(main())

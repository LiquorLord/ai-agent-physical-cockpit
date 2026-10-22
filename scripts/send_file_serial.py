#!/usr/bin/env python3
import os
import sys
import termios
import select
import time
import base64
import hashlib

def send_file(local_path, remote_path, port='/dev/ttyUSB1'):
    with open(local_path, 'rb') as f:
        data = f.read()
    
    local_sha = hashlib.sha256(data).hexdigest()
    print(f"[*] Sending {local_path} ({len(data)} bytes, sha256: {local_sha[:8]}...) to {remote_path}")
    
    fd = os.open(port, os.O_RDWR | os.O_NOCTTY)
    attrs = termios.tcgetattr(fd)
    attrs[0] = 0
    attrs[1] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    attrs[3] = 0
    attrs[4] = termios.B115200
    attrs[5] = termios.B115200
    termios.tcsetattr(fd, termios.TCSANOW, attrs)

    # Helper to send a command and wait for prompt
    def exec_cmd(cmd, timeout=3.0):
        # flush input
        while True:
            r, _, _ = select.select([fd], [], [], 0.05)
            if not r:
                break
            os.read(fd, 1024)
        os.write(fd, f"{cmd}\n".encode())
        res = b""
        start = time.time()
        while time.time() - start < timeout:
            r, _, _ = select.select([fd], [], [], 0.1)
            if r:
                chunk = os.read(fd, 1024)
                if chunk:
                    res += chunk
                    if b"# " in res or b"$ " in res:
                        break
        return res.decode('utf-8', errors='replace')

    # Ensure remote directory exists and clear remote temp file
    remote_tmp = f"{remote_path}.b64"
    exec_cmd(f"rm -f {remote_tmp} {remote_path}")
    
    b64_str = base64.b64encode(data).decode('ascii')
    chunk_size = 512
    total_chunks = (len(b64_str) + chunk_size - 1) // chunk_size
    print(f"[*] Uploading {total_chunks} base64 chunks...")
    
    for i in range(0, len(b64_str), chunk_size):
        chunk = b64_str[i:i+chunk_size]
        exec_cmd(f"echo -n '{chunk}' >> {remote_tmp}", timeout=1.0)
        curr = i // chunk_size + 1
        if curr % 10 == 0 or curr == total_chunks:
            sys.stdout.write(f"\r[*] Progress: {curr}/{total_chunks} chunks ({curr*100//total_chunks}%)")
            sys.stdout.flush()
    print()

    # Decode and sync
    out = exec_cmd(f"base64 -d {remote_tmp} > {remote_path} && rm -f {remote_tmp} && sync", timeout=5.0)
    
    # Verify sha256
    out = exec_cmd(f"sha256sum {remote_path}", timeout=3.0)
    print(f"[*] Remote sha256sum: {out.strip()}")
    if local_sha in out:
        print(f"[+] Verification SUCCESSFUL: SHA256 matches {local_sha}!")
        os.close(fd)
        return True
    else:
        print(f"[-] Verification FAILED! Expected {local_sha}")
        os.close(fd)
        return False

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: send_file_serial.py <local_path> <remote_path>")
        sys.exit(1)
    success = send_file(sys.argv[1], sys.argv[2])
    sys.exit(0 if success else 1)

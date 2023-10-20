import os
import socket

USE_CMUS_SOCKET = os.environ.get("USE_CMUS_SOCKET", False)

def execute_cmus_command(command):
    if USE_CMUS_SOCKET:
        s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        s.connect("/home/pi/.config/cmus/socket")
        s.send(command.encode())
        s.close()
    else:
        os.system(f'cmus-remote {command}')

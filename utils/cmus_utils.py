import os
import socket

USE_CMUS_SOCKET = os.environ.get("USE_CMUS_SOCKET", False)
QUEUE_AND_PLAY_FOLDER = 0
PLAY_PAUSE = 1
NEXT = 2
ACTION_COMMAND_MAP = {
    QUEUE_AND_PLAY_FOLDER: '-s -c -f',
    PLAY_PAUSE: '-u',
    NEXT: '-n'
}

def send_to_cmus_socket(commands):
    cmus_socket = os.path.join(os.environ.get("XDG_RUNTIME_DIR"), 'cmus-socket')
    for cmd in commands:
        print(cmd)
        s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        s.connect(cmus_socket)
        s.send(cmd.encode() + b'\n')
        s.close()

def execute_cmus_command(action, path=None):
    """
    Executes the desired cmus command.

    Parameters:
    - action (str): cmus action such as '-u', '-n' or '-s -c -f'.
    - path (str, optional): Path to the music folder or file if needed for the action.
    """
    if action == QUEUE_AND_PLAY_FOLDER and path:
        if USE_CMUS_SOCKET:
            commands = [
                'player-stop',
                'clear'
            ]
            # For each file in path, add an 'add {}/filename' command
            # For simplicity, let's assume all files in the directory are valid
            # You might need additional error handling in a real-world scenario
            for filename in sorted(os.listdir(path)):
                if filename.endswith('.mp3'):  # or use a more comprehensive filter
                    commands.append(f'add {os.path.join(path, filename)}')
            commands.append('player-play')
            send_to_cmus_socket(commands)
        else:
            cmd = f'cmus-remote {ACTION_COMMAND_MAP[action]} {path}/*'
            os.system(cmd)

# def execute_cmus_command(action, path=None):
#     cmd = ''
#     if action == QUEUE_AND_PLAY_FOLDER and path:
#         cmd = f'{ACTION_COMMAND_MAP[action]} {path}/*'
#     else:
#         cmd = f'{ACTION_COMMAND_MAP[action]}'
# 
#     if USE_CMUS_SOCKET:
#         cmus_socket = os.path.join(
#                 os.environ.get("XDG_RUNTIME_DIR"), 'cmus-socket'
#          )
#         s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
#         s.connect(cmus_socket)
#         s.send(cmd.encode())
#         s.close()
#     else:
#         os.system(f'cmus-remote {cmd}')


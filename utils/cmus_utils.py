import os
import socket
import time

QUEUE_AND_PLAY_FOLDER = 0
PLAY_PAUSE = 1
NEXT = 2
STATUS = 3


def send_to_cmus_socket(commands):
    cmus_socket = os.path.join(
        os.environ.get("XDG_RUNTIME_DIR"),
        "cmus-socket")
    for cmd in commands:
        if cmd != "status":
            print(cmd)
        s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        s.connect(cmus_socket)
        s.send(cmd.encode() + b"\n")
        time.sleep(0.05)
        data = s.recv(4096)
        s.close()
    return data


def music_is_playing():
    try:
        return b"status playing" in send_to_cmus_socket(["status"])
    except BaseException:
        return False


def execute_cmus_command(action, path=None):
    """
    Executes the desired cmus command.

    Parameters:
    - action (str): cmus action such as 'player-pause',
        'player-next' or queue and play commands.
    - path (str, optional): Path to the music folder or
        file if needed for the action.
    """
    if action == QUEUE_AND_PLAY_FOLDER and path:
        commands = ["player-stop", "clear"]
        # For each file in path, add an 'add {}/filename' command
        for filename in sorted(os.listdir(path)):
            if filename.endswith(".mp3"):  # or use a more comprehensive filter
                commands.append(f"add {os.path.join(path, filename)}")
        commands.append("player-next")
        commands.append("player-play")
        send_to_cmus_socket(commands)
    else:
        send_to_cmus_socket([action_to_command(action)])


def action_to_command(action):
    return {
        # Assuming 'player-play' starts playing from the queue
        QUEUE_AND_PLAY_FOLDER: "player-play",
        PLAY_PAUSE: "player-pause",
        NEXT: "player-next",
    }[action]

import os
import socket
import time

QUEUE_AND_PLAY_FOLDER = 0
PLAY_PAUSE = 1
NEXT = 2
STATUS = 3
SHUFFLE = 4
REPEAT = 5


def send_to_cmus_socket(commands):
    try:
        cmus_socket_path = os.path.join(
            os.environ.get("XDG_RUNTIME_DIR", "/run/user/1000"), "cmus-socket"
        )

        if not os.path.exists(cmus_socket_path):
            raise FileNotFoundError(
                f"Socket file '{cmus_socket_path}' not found."
            )

        data = b""
        for cmd in commands:
            if cmd != "status":
                print(f"Sending command: {cmd}")

            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
                s.connect(cmus_socket_path)
                s.send(cmd.encode() + b"\n")
                time.sleep(0.05)
                data = s.recv(4096)

        return data

    except socket.error as e:
        print(f"Socket error occurred: {e}")
    except FileNotFoundError as e:
        print(e)
    except Exception as e:
        print(f"An unexpected error occurred: {e}")
    return None


def ensure_is_cmus_running():
    cmus_socket_path = os.path.join(
        os.environ.get("XDG_RUNTIME_DIR", "/run/user/1000"), "cmus-socket"
    )

    if not os.path.exists(cmus_socket_path):
        print("cmus is not running.")
        raise FileNotFoundError(f"Socket file '{cmus_socket_path}' not found.")


def cmus_status():
    try:
        status_output = send_to_cmus_socket(["status"])
        is_playing = b"status playing" in status_output
        is_shuffle = b"set shuffle true" in status_output
        is_repeat = b"set repeat true" in status_output
        return is_playing, is_shuffle, is_repeat
    except BaseException as e:
        print("Could not get cmus status")
        print(e)
        return False, False, False


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
        SHUFFLE: "set shuffle",
        REPEAT: "set repeat",
    }[action]

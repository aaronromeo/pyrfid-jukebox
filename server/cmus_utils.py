import os
import socket
import time
from logger import Logger

QUEUE_AND_PLAY_FOLDER = 0
PLAY_PAUSE = 1
NEXT = 2
STATUS = 3
SHUFFLE = 4
REPEAT = 5
STOP = 6


def send_to_cmus_socket(commands):
    try:
        cmus_socket_path = os.path.join(
            os.environ.get("XDG_RUNTIME_DIR", "/home/pi"), "cmus-socket"
        )

        if not os.path.exists(cmus_socket_path):
            raise FileNotFoundError(
                f"Socket file '{cmus_socket_path}' not found."
            )

        data = b""
        for cmd in commands:
            if cmd != "status":
                Logger.info(f"Sending command: {cmd}")

            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
                s.connect(cmus_socket_path)
                s.send(cmd.encode() + b"\n")
                time.sleep(0.05)
                data = s.recv(4096)

                Logger.info(f"Completed command: {cmd}")

        return data

    except socket.error as e:
        Logger.critical(f"Socket error occurred: {e}")
        raise e
    except Exception as e:
        Logger.critical(f"An unexpected error occurred: {e}")
        raise e


def ensure_is_cmus_running():
    cmus_socket_path = os.path.join(
        os.environ.get("XDG_RUNTIME_DIR", "/home/pi"), "cmus-socket"
    )

    if not os.path.exists(cmus_socket_path):
        Logger.critical("cmus is not running.")
        raise FileNotFoundError(f"Socket file '{cmus_socket_path}' not found.")

    return True


def cmus_status():
    Logger.info("Requesting cmus status")
    status_output = send_to_cmus_socket(["status"])
    Logger.info(f"Received cmus status {status_output}")
    is_playing = b"status playing" in status_output
    is_shuffle = b"set shuffle true" in status_output
    is_repeat = b"set repeat true" in status_output
    return is_playing, is_shuffle, is_repeat


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
    elif action == SHUFFLE or action == REPEAT:
        cmus_status_position = 1 if action == SHUFFLE else 2
        toggle = not cmus_status()[cmus_status_position]
        send_to_cmus_socket([f"{action_to_command(action)}={toggle}"])
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
        STOP: "player-stop",
    }[action]

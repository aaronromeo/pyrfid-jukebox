from datetime import datetime
import sys
from cmus_utils import (
    execute_cmus_command,
    cmus_status,
    QUEUE_AND_PLAY_FOLDER,
    PLAY_PAUSE,
    NEXT,
    SHUFFLE,
    REPEAT,
)
import os
from mfrc522 import SimpleMFRC522
import RPi.GPIO as GPIO
import threading
import time
import json

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27
LED_PIN = 22

script_dir = os.path.dirname(os.path.abspath(__file__))
RFID_TO_MUSIC_MAP = os.path.join(script_dir, "rfid_map.json")
LOCK_FILE = "/tmp/pyrfid_jukebox.lock"

# Setup GPIO
GPIO.setmode(GPIO.BCM)
GPIO.setup(BUTTON_PLAY_PAUSE, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(BUTTON_NEXT_TRACK, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(LED_PIN, GPIO.OUT)

# Initialize RFID reader
rfid_reader = SimpleMFRC522()

# Process lock


def is_process_running(pid):
    # Check if a process with the given PID is currently running.
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    else:
        return True


def acquire_lock():
    # Acquire the lock if possible, and return the lock file handle.
    lock_file = open(LOCK_FILE, "a+")
    lock_file.seek(0)
    pid_str = lock_file.read().strip()

    # Check if the PID from the lock file is still running
    if pid_str and is_process_running(int(pid_str)):
        print(f"{datetime.now()} - Script is already running.")
        sys.exit(1)
    else:
        # Write the current PID to the lock file
        lock_file.seek(0)
        lock_file.truncate()
        lock_file.write(str(os.getpid()))
        lock_file.flush()
        return lock_file


def ensure_is_cmus_running():
    if cmus_status is None:
        print("cmus is not running")
        sys.exit(1)


def data_to_map(data):
    with open(RFID_TO_MUSIC_MAP, "w") as file:
        json.dump(data, file, indent=4)


####
# Button callback functions


def play_pause_callback(channel):
    print("Play/pause button pressed")
    execute_cmus_command(PLAY_PAUSE)


def next_track_callback(channel):
    print("Next track button pressed")
    execute_cmus_command(NEXT)


def toggle_shuffle_callback(channel):
    print("Toggle shuffle")
    execute_cmus_command(SHUFFLE)


def toggle_repeat_callback(channel):
    print("Toggle repeat")
    execute_cmus_command(REPEAT)


def music_is_playing():
    return cmus_status()[0]


def led_update_loop():
    while not exit_event.is_set():
        if music_is_playing():
            GPIO.output(LED_PIN, GPIO.HIGH)
        else:
            GPIO.output(LED_PIN, GPIO.LOW)
        time.sleep(0.5)  # you can adjust the sleep time as needed


####


# Set up button event detection with debouncing
DEBOUNCE_TIME = 750  # milliseconds
GPIO.add_event_detect(
    BUTTON_PLAY_PAUSE,
    GPIO.FALLING,
    callback=play_pause_callback,
    bouncetime=DEBOUNCE_TIME,
)
GPIO.add_event_detect(
    BUTTON_NEXT_TRACK,
    GPIO.FALLING,
    callback=next_track_callback,
    bouncetime=DEBOUNCE_TIME,
)

# Main loop
try:
    print(f"{datetime.now()} - Script started")

    # Attempt to acquire the lock
    lock_file = acquire_lock()

    # Ensure cmus is running
    ensure_is_cmus_running()

    exit_event = threading.Event()  # this is used to signal the thread to stop

    led_thread = threading.Thread(target=led_update_loop)
    led_thread.start()

    print("Ready to read")
    while True:
        data = {}

        # Create map file if it doesn't exist
        if not os.path.exists(RFID_TO_MUSIC_MAP):
            data_to_map(data)

        try:
            # Check for RFID card
            rfid_id, text = rfid_reader.read()
            print("Received RFID card: %s" % rfid_id)

            update_map = False

            # Load existing map file
            with open(RFID_TO_MUSIC_MAP, "r") as file:
                data = json.load(file)

            folder_path = data.get(rfid_id, "")
            if folder_path:
                folder_path = os.path.abspath(data[rfid_id])

                print("Looking for folder: %s" % folder_path)

                # If folder exists, execute the command
                if os.path.isdir(folder_path):
                    print("Folder found")
                    execute_cmus_command(QUEUE_AND_PLAY_FOLDER, folder_path)
                else:
                    print("Folder not found")

                    if rfid_id in data:
                        update_map = True
                        data[rfid_id] = ""
            else:
                print("RFID ID not in mapping or mapped to an empty path.")

            if update_map:
                data_to_map(data)

        except Exception as e:
            print(f"An error occurred: {e}")

finally:
    exit_event.set()  # signal the led_thread to stop
    led_thread.join()  # wait for the led_thread to finish
    GPIO.cleanup()

    if lock_file:
        lock_file.close()
        os.remove(LOCK_FILE)

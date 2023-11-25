from datetime import datetime
import sys
from cmus_utils import (
    execute_cmus_command,
    ensure_is_cmus_running,
    QUEUE_AND_PLAY_FOLDER,
)
import os
from mfrc522 import SimpleMFRC522
import RPi.GPIO as GPIO
import threading
import json
import warnings
import pyttsx3

from peripheral_helpers import (
    BUTTON_DEBOUNCE_TIME,
    BUTTON_NEXT_TRACK,
    BUTTON_PLAY_PAUSE,
    LED_PIN,
    led_update_loop_factory,
    next_track_callback,
    play_pause_callback,
)

print(f"{datetime.now()} - Script started")
warnings.simplefilter("error")

script_dir = os.path.dirname(os.path.abspath(__file__))
RFID_TO_MUSIC_MAP = os.path.join(script_dir, "rfid_map.json")
LOCK_FILE = "/tmp/pyrfid_jukebox.lock"

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
        raise RuntimeError("Script is already running.")
    else:
        print(f"{datetime.now()} - Aquiring lock file.")
        # Write the current PID to the lock file
        lock_file.seek(0)
        lock_file.truncate()
        lock_file.write(str(os.getpid()))
        lock_file.flush()
        return lock_file


def data_to_map(data):
    with open(RFID_TO_MUSIC_MAP, "w") as file:
        print(f"Writing to map file {RFID_TO_MUSIC_MAP}")
        json.dump(data, file, indent=4)


def speak(text):
    engine = pyttsx3.init()
    engine.say(text)
    engine.runAndWait()


# Set up button event detection with debouncing
has_error = False

# Variables for the final cleanup
exit_event = None
led_thread = None
lock_file = None

# Main loop
try:
    # Setup GPIO
    print("GPIO setup")
    GPIO.setmode(GPIO.BCM)
    GPIO.setup(BUTTON_PLAY_PAUSE, GPIO.IN, pull_up_down=GPIO.PUD_UP)
    GPIO.setup(BUTTON_NEXT_TRACK, GPIO.IN, pull_up_down=GPIO.PUD_UP)
    GPIO.setup(LED_PIN, GPIO.OUT)
    GPIO.add_event_detect(
        BUTTON_PLAY_PAUSE,
        GPIO.FALLING,
        callback=play_pause_callback,
        bouncetime=BUTTON_DEBOUNCE_TIME,
    )
    GPIO.add_event_detect(
        BUTTON_NEXT_TRACK,
        GPIO.FALLING,
        callback=next_track_callback,
        bouncetime=BUTTON_DEBOUNCE_TIME,
    )

    # Initialize RFID reader
    print("Initialize RFID reader")
    rfid_reader = SimpleMFRC522()

    # Attempt to acquire the lock
    lock_file = acquire_lock()

    exit_event = threading.Event()  # this is used to signal the thread to stop

    led_thread = threading.Thread(target=led_update_loop_factory(exit_event))
    led_thread.start()

    print("Ready to read")
    while True:
        # Ensure cmus is running
        ensure_is_cmus_running()

        data = {}

        print(f"Loading map file {RFID_TO_MUSIC_MAP}")

        # Create map file if it doesn't exist
        if not os.path.exists(RFID_TO_MUSIC_MAP):
            data_to_map(data)

        try:
            # Check for RFID card
            rfid_id, text = rfid_reader.read()
            rfid_id = str(rfid_id)
            print("Received RFID card: %s" % rfid_id)

            update_map = False

            # Load existing map file
            with open(RFID_TO_MUSIC_MAP, "r") as file:
                data = json.load(file)

            print(data)
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
                    speak("Folder not found")

                    # # Resetting the data value since the folder is not found
                    # update_map = True
                    # data[rfid_id] = ""
            else:
                print("RFID ID not in mapping or mapped to an empty path.")

                update_map = True
                data[rfid_id] = ""

            if update_map:
                data_to_map(data)
        except Exception as e:
            print(f"Error during RFID read or processing: {e}")
            raise e

except KeyboardInterrupt:
    print("Script interrupted by user")

except Exception as e:
    print(f"Unhandled exception: {e}")
    has_error = True

finally:
    if exit_event is not None:
        print("signal the led_thread to stop")
        exit_event.set()  # signal the led_thread to stop

    if led_thread is not None:
        print("wait for the led_thread to finish")
        led_thread.join()  # wait for the led_thread to finish

    print("GPIO cleanup")
    GPIO.cleanup()

    if lock_file:
        print("Removing lock file")
        lock_file.close()
        os.remove(LOCK_FILE)

    if has_error:
        print("Exiting with error")
        sys.exit(1)

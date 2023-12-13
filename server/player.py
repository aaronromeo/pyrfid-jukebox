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
from logger import Logger

from peripheral_helpers import (
    blink_play,
    blink_leds_row_once,
    blink_red_leds_once,
    led_update_loop_factory,
    add_button_detections,
    led_setup,
    button_setup,
)

Logger.info("Script started")
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
        Logger.critical("Script is already running.")
        raise RuntimeError("Script is already running.")
    else:
        Logger.info("Aquiring lock file.")
        # Write the current PID to the lock file
        lock_file.seek(0)
        lock_file.truncate()
        lock_file.write(str(os.getpid()))
        lock_file.flush()
        return lock_file


def data_to_map(data):
    with open(RFID_TO_MUSIC_MAP, "w") as file:
        Logger.info(f"Writing to map file {RFID_TO_MUSIC_MAP}")
        json.dump(data, file, indent=4)


# Set up button event detection with debouncing
has_error = False

# Variables for the final cleanup
exit_event = None
led_thread = None
lock_file = None

# Main loop
try:
    # Setup GPIO
    Logger.info("Starting GPIO setup")
    GPIO.setmode(GPIO.BCM)

    # The current connections are between the GPIO pin and GND when closed.
    # This means the GPIO will read LOW when the button is pressed
    # Adding in the `GPIO.PUD_UP` adds an Internal Pull-Up Resistor
    button_setup()
    led_setup()

    # Related to the comment above about the GPIO pin reading LOW when the
    # button is pressed. In this case, the event detected is FALLING (detecting
    # a HIGH to LOW) as the first button pressed action. An alternative is to
    # detect a RISING event (LOW to HIGH) which would occur when the button is
    # released. Another alternative is to detect BOTH.
    add_button_detections()

    # Initialize RFID reader
    Logger.info("Initialize RFID reader")
    rfid_reader = SimpleMFRC522()

    # Attempt to acquire the lock
    lock_file = acquire_lock()

    exit_event = threading.Event()  # this is used to signal the thread to stop

    led_thread = threading.Thread(target=led_update_loop_factory(exit_event))
    led_thread.start()

    played_ready_message = False
    # Ensure cmus is running and LED thread is alive
    while ensure_is_cmus_running() and led_thread.is_alive():
        if not played_ready_message:
            Logger.info("Ready to read")
            played_ready_message = True

        data = {}

        Logger.info(f"Loading map file {RFID_TO_MUSIC_MAP}")

        # Create map file if it doesn't exist
        if not os.path.exists(RFID_TO_MUSIC_MAP):
            data_to_map(data)

        try:
            # Check for RFID card
            rfid_id, text = rfid_reader.read()
            rfid_id = str(rfid_id)
            Logger.info("Received RFID card: %s" % rfid_id)

            update_map = False

            # Load existing map file
            with open(RFID_TO_MUSIC_MAP, "r") as file:
                data = json.load(file)

            Logger.debug(data)
            folder_path = data.get(rfid_id, "")
            if folder_path:
                folder_path = os.path.abspath(data[rfid_id])

                Logger.info("Looking for folder: %s" % folder_path)

                # If folder exists, execute the command
                if os.path.isdir(folder_path):
                    Logger.info("Folder found")
                    blink_leds_row_once()
                    blink_play(30)
                    execute_cmus_command(QUEUE_AND_PLAY_FOLDER, folder_path)
                else:
                    Logger.warning(f"Folder '{folder_path}' not found")
                    blink_red_leds_once()
                    blink_red_leds_once()

                    # # Resetting the data value since the folder is not found
                    # update_map = True
                    # data[rfid_id] = ""
            else:
                Logger.warning(
                    "RFID ID not in mapping or mapped to an empty path."
                )
                blink_red_leds_once()
                blink_red_leds_once()

                update_map = True
                data[rfid_id] = ""

            if update_map:
                data_to_map(data)
        except Exception as e:
            Logger.critical(f"Error during RFID read or processing: {e}")
            raise e

except KeyboardInterrupt:
    Logger.critical("Script interrupted by user")

except Exception as e:
    Logger.critical(f"Unhandled exception: {e}")
    has_error = True

finally:
    if exit_event is not None:
        Logger.info("signal the led_thread to stop")
        exit_event.set()  # signal the led_thread to stop

    if led_thread is not None:
        Logger.info("wait for the led_thread to finish")
        led_thread.join()  # wait for the led_thread to finish

    Logger.info("GPIO cleanup initiated")
    GPIO.cleanup()

    if lock_file:
        Logger.info("Removing lock file")
        lock_file.close()
        os.remove(LOCK_FILE)

    if has_error:
        Logger.info("Exiting with error")
        sys.exit(1)

from cmus_utils import (
    execute_cmus_command,
    music_is_playing,
    QUEUE_AND_PLAY_FOLDER,
    PLAY_PAUSE,
    NEXT,
)
import os
from mfrc522 import SimpleMFRC522
import RPi.GPIO as GPIO
import threading
import time

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27
LED_PIN = 22

# Setup GPIO
GPIO.setmode(GPIO.BCM)
GPIO.setup(BUTTON_PLAY_PAUSE, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(BUTTON_NEXT_TRACK, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(LED_PIN, GPIO.OUT)

# Initialize RFID reader
rfid_reader = SimpleMFRC522()

# Button callback functions


def play_pause_callback(channel):
    print("Play/pause button pressed")
    execute_cmus_command(PLAY_PAUSE)


def next_track_callback(channel):
    print("Next track button pressed")
    execute_cmus_command(NEXT)


def led_update_loop():
    while not exit_event.is_set():
        if music_is_playing():
            GPIO.output(LED_PIN, GPIO.HIGH)
        else:
            GPIO.output(LED_PIN, GPIO.LOW)
        time.sleep(0.5)  # you can adjust the sleep time as needed


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
    exit_event = threading.Event()  # this is used to signal the thread to stop
    led_thread = threading.Thread(target=led_update_loop)
    led_thread.start()

    print("Ready to read")
    while True:
        # Check for RFID card
        rfid_id, text = rfid_reader.read()
        folder_path = os.path.abspath(
            os.path.join("music", "card-%s" % rfid_id)
        )
        print("Received RFID card: %s" % rfid_id)
        print("Looking for folder: %s" % folder_path)

        if os.path.exists(folder_path):
            print("Folder found")
            execute_cmus_command(QUEUE_AND_PLAY_FOLDER, folder_path)

finally:
    exit_event.set()  # signal the led_thread to stop
    led_thread.join()  # wait for the led_thread to finish
    GPIO.cleanup()

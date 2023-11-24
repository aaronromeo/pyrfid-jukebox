import RPi.GPIO as GPIO
import time
from cmus_utils import (
    execute_cmus_command,
    cmus_status,
    PLAY_PAUSE,
    NEXT,
    SHUFFLE,
    REPEAT,
)

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27

BUTTON_DEBOUNCE_TIME = 750  # milliseconds
LED_PIN = 22


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


def led_update_loop_factory(exit_event):
    def led_update_loop():
        while not exit_event.is_set():
            if music_is_playing():
                GPIO.output(LED_PIN, GPIO.HIGH)
            else:
                GPIO.output(LED_PIN, GPIO.LOW)
            time.sleep(0.5)  # you can adjust the sleep time as needed

    return led_update_loop

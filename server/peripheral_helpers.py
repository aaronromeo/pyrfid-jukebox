import RPi.GPIO as GPIO
import time
from cmus_utils import (
    execute_cmus_command,
    cmus_status,
    PLAY_PAUSE,
    NEXT,
    SHUFFLE,
    REPEAT,
    STOP,
)

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27
BUTTON_STOP_TRACK = 18
BUTTON_REPEAT_TRACK = 15
BUTTON_SHUFFLE_TRACK = 14

BUTTON_DEBOUNCE_TIME = 750  # milliseconds
PLAY_LED_PIN = 22
REPEAT_LED_PIN = 5
SHUFFLE_LED_PIN = 6


def play_pause_callback(channel):
    print("Play/pause button pressed")
    execute_cmus_command(PLAY_PAUSE)


def next_track_callback(channel):
    print("Next track button pressed")
    execute_cmus_command(NEXT)


def stop_track_callback(channel):
    print("Stop track button pressed")
    execute_cmus_command(STOP)


def toggle_shuffle_callback(channel):
    print("Toggle shuffle")
    execute_cmus_command(SHUFFLE)


def toggle_repeat_callback(channel):
    print("Toggle repeat")
    execute_cmus_command(REPEAT)


def music_is_playing():
    return cmus_status()[0]


def music_is_shuffling():
    return cmus_status()[1]


def music_is_repeating():
    return cmus_status()[2]


def blink_red_leds_once():
    GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
    GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
    time.sleep(0.5)  # LED is on for 0.5 seconds
    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)
    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)
    time.sleep(0.5)  # LED is off for 0.5 seconds


def blink_leds_row_once():
    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
    time.sleep(0.2)
    GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
    time.sleep(0.2)
    GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
    time.sleep(0.2)
    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)
    time.sleep(0.2)
    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)
    time.sleep(0.2)
    GPIO.output(PLAY_LED_PIN, GPIO.LOW)
    time.sleep(0.2)


def led_update_loop_factory(exit_event):
    def led_update_loop():
        while not exit_event.is_set():
            if music_is_playing():
                GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
                time.sleep(0.5)  # LED is on for 0.5 seconds
                GPIO.output(PLAY_LED_PIN, GPIO.LOW)
                time.sleep(0.5)  # LED is off for 0.5 seconds
            else:
                GPIO.output(PLAY_LED_PIN, GPIO.HIGH)

            if music_is_shuffling():
                GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
            else:
                GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)

            if music_is_repeating():
                GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
            else:
                GPIO.output(REPEAT_LED_PIN, GPIO.LOW)

            time.sleep(0.5)  # you can adjust the sleep time as needed

    return led_update_loop

import RPi.GPIO as GPIO
import time
from cmus_utils import (
    execute_cmus_command,
    ensure_is_cmus_running,
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
BUTTON_REPEAT_TRACK = 16
BUTTON_SHUFFLE_TRACK = 14

BUTTON_DEBOUNCE_TIME = 250  # milliseconds
PLAY_LED_PIN = 22
REPEAT_LED_PIN = 5
SHUFFLE_LED_PIN = 6


def play_pause_callback(channel):
    # Wait for a short period to filter out noise
    time.sleep(BUTTON_DEBOUNCE_TIME)

    # Check if the button is still pressed after the debounce time
    # Assuming a pull-up resistor configuration
    if GPIO.input(BUTTON_PLAY_PAUSE) == GPIO.LOW:
        print("Play/pause button pressed")
        execute_cmus_command(PLAY_PAUSE)


def next_track_callback(channel):
    # Wait for a short period to filter out noise
    time.sleep(BUTTON_DEBOUNCE_TIME)

    # Check if the button is still pressed after the debounce time
    # Assuming a pull-up resistor configuration
    if GPIO.input(BUTTON_NEXT_TRACK) == GPIO.LOW:
        print("Next track button pressed")
        execute_cmus_command(NEXT)


def stop_track_callback(channel):
    # Wait for a short period to filter out noise
    time.sleep(BUTTON_DEBOUNCE_TIME)

    # Check if the button is still pressed after the debounce time
    # Assuming a pull-up resistor configuration
    if GPIO.input(BUTTON_STOP_TRACK) == GPIO.LOW:
        print("Stop track button pressed")
        execute_cmus_command(STOP)


def toggle_shuffle_callback(channel):
    # Wait for a short period to filter out noise
    time.sleep(BUTTON_DEBOUNCE_TIME)

    # Check if the button is still pressed after the debounce time
    # Assuming a pull-up resistor configuration
    if GPIO.input(BUTTON_SHUFFLE_TRACK) == GPIO.LOW:
        print("Toggle shuffle")
        execute_cmus_command(SHUFFLE)


def toggle_repeat_callback(channel):
    # Wait for a short period to filter out noise
    time.sleep(BUTTON_DEBOUNCE_TIME)

    # Check if the button is still pressed after the debounce time
    # Assuming a pull-up resistor configuration
    if GPIO.input(BUTTON_REPEAT_TRACK) == GPIO.LOW:
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
    time.sleep(0.2)  # LED is on for 0.5 seconds
    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)
    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)
    time.sleep(0.2)  # LED is off for 0.5 seconds


def blink_leds_row_once():
    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
    time.sleep(0.3)
    GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
    time.sleep(0.3)
    GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
    time.sleep(0.5)
    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)
    time.sleep(0.3)
    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)
    time.sleep(0.3)
    GPIO.output(PLAY_LED_PIN, GPIO.LOW)
    time.sleep(0.2)


def led_update_loop_factory(exit_event):
    def led_update_loop():
        while not exit_event.is_set():
            try:
                if music_is_playing():
                    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
                    time.sleep(0.5)  # LED is on for 0.5 seconds
                    GPIO.output(PLAY_LED_PIN, GPIO.LOW)
                    time.sleep(0.5)  # LED is off for 0.5 seconds
                elif ensure_is_cmus_running():
                    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(PLAY_LED_PIN, GPIO.LOW)

                if music_is_shuffling():
                    GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)

                if music_is_repeating():
                    GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)

                time.sleep(0.5)  # you can adjust the sleep time as needed
            except Exception as e:
                print(f"Error in LED Loop: {e}")
                raise e

    return led_update_loop

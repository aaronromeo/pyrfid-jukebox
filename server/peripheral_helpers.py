# import inspect
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
from logger import Logger

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27
BUTTON_STOP_TRACK = 18
BUTTON_REPEAT_TRACK = 13
BUTTON_SHUFFLE_TRACK = 19

BUTTON_DEBOUNCE_TIME = 250  # milliseconds
PLAY_LED_PIN = 22
REPEAT_LED_PIN = 5
SHUFFLE_LED_PIN = 6


def high_check(pin):
    time.sleep(0.01)
    if GPIO.input(pin) == GPIO.HIGH:
        return True
    else:
        return False


def play_pause_callback(pin):
    if not high_check(pin):
        return True

    Logger.info("Play/pause button pressed")
    execute_cmus_command(PLAY_PAUSE)


def next_track_callback(pin):
    if not high_check(pin):
        return True

    Logger.info("Next track button pressed")
    execute_cmus_command(NEXT)


def stop_track_callback(pin):
    if not high_check(pin):
        return True

    Logger.info("Stop track button pressed")
    execute_cmus_command(STOP)


def toggle_shuffle_callback(pin):
    if not high_check(pin):
        return True
    Logger.info("Toggle shuffle")
    execute_cmus_command(SHUFFLE)
    for x in range(20):
        GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
        time.sleep(0.1)
        GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)
        time.sleep(0.1)


def toggle_repeat_callback(pin):
    if not high_check(pin):
        return True

    Logger.info("Toggle repeat")
    execute_cmus_command(REPEAT)
    for x in range(20):
        GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
        time.sleep(0.1)
        GPIO.output(REPEAT_LED_PIN, GPIO.LOW)
        time.sleep(0.1)


BUTTON_TO_FUNCTION_MAP = {
    BUTTON_REPEAT_TRACK: toggle_repeat_callback,
    BUTTON_SHUFFLE_TRACK: toggle_shuffle_callback,
    BUTTON_PLAY_PAUSE: play_pause_callback,
    BUTTON_NEXT_TRACK: next_track_callback,
    BUTTON_STOP_TRACK: stop_track_callback,
}


def add_button_detections():
    for button in BUTTON_TO_FUNCTION_MAP:
        Logger.info(f"Removing button detection on {button}")
        GPIO.remove_event_detect(button)
        Logger.info(f"Add button detection on {button}")
        GPIO.add_event_detect(
            button,
            GPIO.RISING,
            callback=BUTTON_TO_FUNCTION_MAP[button],
            bouncetime=BUTTON_DEBOUNCE_TIME,
        )


def button_setup():
    for button in BUTTON_TO_FUNCTION_MAP:
        Logger.info(f"Setting up button {button}")
        GPIO.setup(button, GPIO.IN, pull_up_down=GPIO.PUD_UP)


def led_setup():
    Logger.info(f"Setting up led {PLAY_LED_PIN}")
    GPIO.setup(PLAY_LED_PIN, GPIO.OUT)
    Logger.info(f"Setting up led {SHUFFLE_LED_PIN}")
    GPIO.setup(SHUFFLE_LED_PIN, GPIO.OUT)
    Logger.info(f"Setting up led {REPEAT_LED_PIN}")
    GPIO.setup(REPEAT_LED_PIN, GPIO.OUT)


def music_is_playing():
    return cmus_status()[0]


def music_is_shuffling():
    return cmus_status()[1]


def music_is_repeating():
    return cmus_status()[2]


def blink_play(times):
    for x in range(times):
        GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
        time.sleep(0.1)
        GPIO.output(PLAY_LED_PIN, GPIO.LOW)
        time.sleep(0.1)


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
            # print(f"LED update loop 18 {GPIO.input(18)} ")
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
                Logger.error(f"Error in LED Loop: {e}")
                raise e

    return led_update_loop

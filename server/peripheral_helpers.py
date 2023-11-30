from gpiozero import Button, LED
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
BUTTON_DEBOUNCE_TIME = 0.25  # seconds
PLAY_LED_PIN = 22
REPEAT_LED_PIN = 5
SHUFFLE_LED_PIN = 6

play_led = LED(PLAY_LED_PIN)
shuffle_led = LED(SHUFFLE_LED_PIN)
repeat_led = LED(REPEAT_LED_PIN)


def play_pause_callback():
    print("Play/pause button pressed")
    execute_cmus_command(PLAY_PAUSE)


def next_track_callback():
    print("Next track button pressed")
    execute_cmus_command(NEXT)


def stop_track_callback():
    print("Stop track button pressed")
    execute_cmus_command(STOP)


def toggle_shuffle_callback():
    print("Toggle shuffle")
    execute_cmus_command(SHUFFLE)


def toggle_repeat_callback():
    print("Toggle repeat")
    execute_cmus_command(REPEAT)


def music_is_playing():
    return cmus_status()[0]


def music_is_shuffling():
    return cmus_status()[1]


def music_is_repeating():
    return cmus_status()[2]


def init_buttons():
    btn_play_pause = Button(
        BUTTON_PLAY_PAUSE, bounce_time=BUTTON_DEBOUNCE_TIME, pull_up=True
    )
    btn_play_pause.when_pressed = play_pause_callback

    btn_next = Button(
        BUTTON_NEXT_TRACK, bounce_time=BUTTON_DEBOUNCE_TIME, pull_up=True
    )
    btn_next.when_pressed = next_track_callback

    btn_stop = Button(
        BUTTON_STOP_TRACK, bounce_time=BUTTON_DEBOUNCE_TIME, pull_up=True
    )
    btn_stop.when_pressed = stop_track_callback

    btn_repeat = Button(
        BUTTON_REPEAT_TRACK, bounce_time=BUTTON_DEBOUNCE_TIME, pull_up=True
    )
    btn_repeat.when_pressed = toggle_repeat_callback

    btn_shuffle = Button(
        BUTTON_SHUFFLE_TRACK, bounce_time=BUTTON_DEBOUNCE_TIME, pull_up=True
    )
    btn_shuffle.when_pressed = toggle_shuffle_callback


def blink_red_leds_once():
    shuffle_led.on()
    repeat_led.on()
    time.sleep(0.2)  # LED is on for 0.5 seconds
    shuffle_led.off()
    repeat_led.off()
    time.sleep(0.2)  # LED is off for 0.5 seconds


def blink_leds_row_once():
    play_led.on()
    time.sleep(0.3)
    shuffle_led.on()
    time.sleep(0.3)
    repeat_led.on()
    time.sleep(0.5)
    repeat_led.off()
    time.sleep(0.3)
    shuffle_led.off()
    time.sleep(0.3)
    play_led.off()
    time.sleep(0.2)


def led_update_loop_factory(exit_event):
    def led_update_loop():
        while not exit_event.is_set():
            try:
                if music_is_playing():
                    play_led.blink()
                elif ensure_is_cmus_running():
                    play_led.on()
                else:
                    play_led.off()

                if music_is_shuffling():
                    shuffle_led.on()
                else:
                    shuffle_led.off()

                if music_is_repeating():
                    repeat_led.on()
                else:
                    repeat_led.off()

                time.sleep(0.5)  # you can adjust the sleep time as needed
            except Exception as e:
                print(f"Error in LED Loop: {e}")
                raise e

    return led_update_loop

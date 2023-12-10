import RPi.GPIO as GPIO
import time

# Replace with your actual GPIO pin number
BUTTON_PINS = [
        13,
        19
]

# Set up the GPIO using BCM numbering
GPIO.setmode(GPIO.BCM)

# Set up the GPIO pin as an input. Assuming you have an external pull-down resistor.
for pin in BUTTON_PINS:
    GPIO.setup(pin, GPIO.IN)

last_state = {}

def toggle_callback(pin):
    current_state = GPIO.input(pin)
    if  not (pin in last_state) or last_state[pin] != current_state:
        last_state[pin] = current_state
        timestamp = time.time()
        state_str = 'High' if current_state else 'Low'
        print(f"Time: {timestamp} - Button {pin}State: {state_str}")

try:
    # Read the state of the GPIO pin
    for pin in BUTTON_PINS:
        GPIO.add_event_detect(
            pin,
            GPIO.BOTH,
            callback=toggle_callback,
        )
    while True:
        True

except KeyboardInterrupt:
    # Clean up the GPIO on CTRL+C exit
    GPIO.cleanup()

# Clean up the GPIO on normal exit
GPIO.cleanup()


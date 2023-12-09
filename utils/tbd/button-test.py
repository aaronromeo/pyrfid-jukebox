import RPi.GPIO as GPIO
import time

# Replace with your actual GPIO pin number
BUTTON_PIN = 17

# Set up the GPIO using BCM numbering
GPIO.setmode(GPIO.BCM)

# Set up the GPIO pin as an input. Assuming you have an external pull-down resistor.
GPIO.setup(BUTTON_PIN, GPIO.IN)

last_state = None

try:
    while True:
        # Read the state of the GPIO pin
        current_state = GPIO.input(BUTTON_PIN)

        # Check if the state has changed
        if current_state != last_state:
            last_state = current_state
            timestamp = time.time()
            state_str = 'High' if current_state else 'Low'
            print(f"Time: {timestamp} - Button State: {state_str}")

        # Wait a little while before checking again
        time.sleep(0.01)  # 10ms for debouncing

except KeyboardInterrupt:
    # Clean up the GPIO on CTRL+C exit
    GPIO.cleanup()

# Clean up the GPIO on normal exit
GPIO.cleanup()


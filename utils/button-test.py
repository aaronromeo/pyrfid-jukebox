import RPi.GPIO as GPIO # Import Raspberry Pi GPIO library

GPIO.setwarnings(False) # Ignore warning for now
GPIO.setmode(GPIO.BOARD) # Use physical pin numbering
GPIO.setup(10, GPIO.IN, pull_up_down=GPIO.PUD_DOWN) # Set pin 10 to be an input pin and set initial value to be pulled low (off)

def button_callback(channel):
    print("Button 10 was pushed!")

try:
    # while True: # Run forever
    #     if GPIO.input(10) == GPIO.HIGH:
    #         print("Button 10 was pushed!")
    GPIO.add_event_detect(10,GPIO.RISING,callback=button_callback) # Setup event on pin 10 rising edge

    input("Ready!\n")
except Exception:
    print("Error with %s" % path)      
finally:
    GPIO.cleanup()


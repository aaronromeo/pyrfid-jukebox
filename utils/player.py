import RPi.GPIO as GPIO
import os
import socket
import time
from mfrc522 import SimpleMFRC522
from cmus_utils import execute_cmus_command

# GPIO pin numbers
BUTTON_PLAY_PAUSE = 17
BUTTON_NEXT_TRACK = 27
LED_PIN = 22

# Determine mode of operation from environment variable
USE_CMUS_SOCKET = os.environ.get("USE_CMUS_SOCKET", False)

# Setup GPIO
GPIO.setmode(GPIO.BCM)
GPIO.setup(BUTTON_PLAY_PAUSE, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(BUTTON_NEXT_TRACK, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(LED_PIN, GPIO.OUT)

# Initialize RFID reader
rfid_reader = SimpleMFRC522()

def music_is_playing():
    if USE_CMUS_SOCKET:
        try:
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect("/home/pi/.config/cmus/socket")
            s.send(b"status\n")
            time.sleep(0.05)
            data = s.recv(4096)
            s.close()
            return b'status playing' in data
        except:
            return False
    else:
        return os.system('cmus-remote -Q | grep -q "status playing"') == 0

# Button callback functions
def play_pause_callback(channel):
    print("Play/pause button pressed")
    execute_cmus_command('-u')

def next_track_callback(channel):
    print("Next track button pressed")
    execute_cmus_command('-n')

# Set up button event detection with debouncing
DEBOUNCE_TIME = 200  # 200 milliseconds
GPIO.add_event_detect(BUTTON_PLAY_PAUSE, GPIO.FALLING, callback=play_pause_callback, bouncetime=DEBOUNCE_TIME)
GPIO.add_event_detect(BUTTON_NEXT_TRACK, GPIO.FALLING, callback=next_track_callback, bouncetime=DEBOUNCE_TIME)

# Main loop
try:
    print("Ready to read")
    while True:
        # Check for RFID card
        rfid_id, text = rfid_reader.read()
        folder_path = f"../music/card-{rfid_id}"
        print("Received RFID card: %s" % rfid_id)
        print("Looking for folder: %s" % folder_path)
        
        if os.path.exists(folder_path):
            print("Folder found")
            execute_cmus_command(f'-C "player-play {folder_path}"')

        # Update LED
        if music_is_playing():
            GPIO.output(LED_PIN, GPIO.HIGH)
        else:
            GPIO.output(LED_PIN, GPIO.LOW)

finally:
    GPIO.cleanup()

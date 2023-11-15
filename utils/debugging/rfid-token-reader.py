#!/usr/bin/env python

import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522

reader = SimpleMFRC522()

try:
    print("Ready to read RFID tag")
    id, text = reader.read()
    print("RFID tag detected")
    print("ID:", id)
    print("Text:", text)
except Exception as e:
    print("Error:", e)
finally:
    GPIO.cleanup()

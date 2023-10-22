#!/usr/bin/env python

import os
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522

ROOT_DIR = "music"

reader = SimpleMFRC522()

try:
    print("Ready to read")
    id, _text = reader.read()
    path = os.path.join(ROOT_DIR, "card-%s" % str(id))
    os.mkdir(path)
    print("Create %s" % path)
finally:
    GPIO.cleanup()

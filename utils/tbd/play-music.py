#!/usr/bin/env python

import os
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522

# cmus-remote -s -c -f ~/workspace/pyrfid-jukebox/music/card-1061668068229/*

ROOT_DIR = "music"
CMS_PLAY_FOLDER = "cmus-remote -s -c -f {}"

reader = SimpleMFRC522()

while True:
    try:
        print("Ready to read")
        id, _text = reader.read()
        path = os.path.abspath(os.path.join(ROOT_DIR, "card-%s" % str(id)))
        cmd = CMS_PLAY_FOLDER.format(path)
        print(cmd)
        if os.system(cmd) == 0:
            print("Playing %s" % path)
        else:
            print("Error with %s" % path)
    except Exception:
        print("Error with %s" % path)
    finally:
        GPIO.cleanup()

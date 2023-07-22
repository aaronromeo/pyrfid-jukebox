#!/usr/bin/env python

import os
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522

ROOT_DIR = "music"
CMS_CLEAR_PLAYLIST = "cmus-remote -c -q"
CMS_CLEAR_QUEUE = "cmus-remote -c -p"
CMS_ADD_TO_QUEUE = "cmus-remote -C \"add -q {}\""
CMS_NEXT = "cmus-remote -n"
CMS_PLAY = "cmus-remote -p"

reader = SimpleMFRC522()

while True:
    try:
            print("Ready to read")
            id, _text = reader.read()
            path = os.path.abspath(
                    os.path.join(ROOT_DIR, "card-%s" % str(id))
            )
            cmd = " && ".join([
            CMS_CLEAR_PLAYLIST, CMS_CLEAR_PLAYLIST, CMS_ADD_TO_QUEUE.format(path), CMS_NEXT, CMS_PLAY     
            ])
            print(cmd)
            if os.system(cmd) == 0:
                print("Playing %s" % path)
            else:
                print("Error with %s" % path)
    except Exception:
            print("Error with %s" % path)      
    finally:
            GPIO.cleanup()


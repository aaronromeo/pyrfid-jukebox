import time

DEFAULT_POLL_INTERVAL = 0.05


def poll_for_card(
    reader, poll_interval=DEFAULT_POLL_INTERVAL, sleep=time.sleep
):
    """Return (id, text) once a card is present, sleeping between polls.

    ``reader`` must expose ``read_no_block()``; a falsy id means no card.
    Sleeping between polls keeps the process off the CPU while idle, unlike
    ``SimpleMFRC522.read()`` which spins on SPI with no pause.
    """
    while True:
        rfid_id, text = reader.read_no_block()
        if rfid_id:
            return rfid_id, text
        sleep(poll_interval)

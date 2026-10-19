from rfid_reader import poll_for_card


class FakeReader:
    def __init__(self, results):
        self._results = list(results)
        self.calls = 0

    def read_no_block(self):
        self.calls += 1
        return self._results.pop(0)


def test_returns_first_detected_card():
    reader = FakeReader([(None, None), (None, None), (42, "hello")])
    sleeps = []

    result = poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append)

    assert result == (42, "hello")
    assert reader.calls == 3


def test_sleeps_between_empty_polls():
    reader = FakeReader([(None, None), (None, None), (7, "x")])
    sleeps = []

    poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append)

    assert sleeps == [0.05, 0.05]


def test_does_not_sleep_after_detecting():
    reader = FakeReader([(1, "a")])
    sleeps = []

    assert poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append) == (
        1,
        "a",
    )
    assert sleeps == []

from cmus_utils import execute_cmus_command
from flask import Flask

app = Flask(__name__)

@app.route("/")
def index():
    # You will need to implement a way to get the current playing track
    current_track = get_current_track()
    return f"Currently playing: {current_track}"

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=80)

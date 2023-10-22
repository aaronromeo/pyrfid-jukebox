# RFID-Music

## Getting Started

### Prerequisites
_Assumes `cmus` is installed_
```
sudo apt-get update
sudo apt-get install python3 python3-pip cmus
pip3 install RPi.GPIO mfrc522 flask
```

#### Random helpers
* Create a venv
    * `python -m venv env`
* Install the `requirements.txt`
    * `pip3 install -r requirements.txt`
* Activate venv
    * `source env/bin/activate`
* Dump requirements
    * `pip3 freeze > requirements.txt`

#### GPIO config
![RP4](docs/Screen%20Shot%202023-10-19%20at%2010.07.16%20PM.png)  

Connections for the SDA, SCK, MOSI, MISO, RST of the MFRC522 to the Raspberry Pi
* SDA (Serial Data) connected to GPIO8 (CE0) - Pin 24
* SCK (Serial Clock) connected to GPIO11 (SCLK) - Pin 23
* MOSI (Master Out Slave In) connected to GPIO10 (MOSI) - Pin 19
* MISO (Master In Slave Out) connected to GPIO9 (MISO) - Pin 21
* RST (Reset) connected to any GPIO (e.g., GPIO25) - Pin 22 

Additional buttons
* Play/Pause Button: Connected to GPIO17 - Pin 11.
* Next Track Button: Connected to GPIO27 - Pin 13.
* LED connected to GPIO22 - Pin 15.



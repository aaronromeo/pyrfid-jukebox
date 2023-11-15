# PyRFID Jukebox

## What does this tool do?

## Getting Started

### Setting up the Raspberry Pi Zero W for the Project

### 1. Initial Configuration

- Flash a fresh copy of Raspberry Pi OS (Debian-based) onto a microSD card.
  - I've used the one without the desktop.
  - Using [Raspberry Pi Imager](https://www.raspberrypi.com/software/) allows the setup of SSH

### 2. Deb Package Installations

```bash
sudo apt-get -y update
sudo apt-get -y upgrade
sudo apt-get install -y openssh-server python3 python3-pip cmus bluetooth bluez pi-bluetooth nodejs npm yarn
sudo reboot now
sudo apt-get install -y git vim tmux
sudo apt-get install pulseaudio*
```

### 3. Bluetooth Configuration

- Add user `pi` to the `lp` group for Bluetooth permissions:

  ```bash
  sudo usermod -a -G lp pi
  ```

- Once installed, follow these next steps to pair with your Bluetooth keyboard:

  1. Run the Bluetooth program by typing `bluetoothctl`.

  2. Turn on the Bluetooth, if not already on, by typing `power on`.

  3. Enter device discovery mode with `scan on` command if device is not yet listed in devices.

  4. Turn the agent on with `agent on`.

  5. Enter `pair <MAC Address>` to do the pairing between devices.

  6. You may be prompted to enter a passcode on the Bluetooth keyboard; if so, type this on the keyboard and press enter.

  7. You will need to add the device to a list of trusted devices with `trust <MAC Address>`.

  8. Finally, connect to your device with `connect <MAC Address>`.

  Note: For a list of Bluetooth commands type `help` in the command line.

### 4. SSH Configuration for Stability

- Backup the SSH configuration:

  ```bash
  sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup
  ```

- Edit the SSH configuration:

  ```bash
  sudo nano /etc/ssh/sshd_config
  ```

- Add the following line to prevent SSH from becoming nonresponsive:

  ```txt
  IPQoS cs0 cs0
  ```

- Restart the SSH service to apply changes:

  ```bash
  sudo service ssh restart
  ```

### 5. Establishing SSH Connection Using Keys

1. On your **local machine**, generate an SSH key pair:

   ```bash
   ssh-keygen
   ```

2. Copy the public key to the Raspberry Pi:

   ```bash
   ssh-copy-id pi@<RaspberryPi_IP_Address>
   ```

3. Now you can SSH into your Raspberry Pi without entering a password.

### 6. Bluetooth Auto-connect on Reboot

To automatically establish a Bluetooth connection on reboot, the following script named `btconnect.sh` is present in the home directory:

```bash
#!/bin/bash
pulseaudio --start
bluetoothctl power on
bluetoothctl connect <MAC Address>
paplay -p --device=1 /usr/share/sounds/alsa/Front_Center.wav
```

This script is executed at reboot using a crontab entry:

```bash
@reboot rm /home/pi/btconnect.log && sleep 10 && /home/pi/btconnect.sh > /home/pi/btconnect.log 2>&1
```

### 7. Project setup

- Create a venv
  - `python -m venv env`
- Activate venv
  - `source env/bin/activate`
- Install the `requirements.txt`
  - `pip3 install -r requirements.txt`

- Dump requirements
  - `pip3 freeze > requirements.txt`

## GPIO config

### RP4 Pinout

![RP4](docs/RP4-pinout.png)

#### RP4 - Connections for the SDA, SCK, MOSI, MISO, RST of the MFRC522

- SDA (Serial Data) connected to GPIO8 (CE0) - Pin 24
- SCK (Serial Clock) connected to GPIO11 (SCLK) - Pin 23
- MOSI (Master Out Slave In) connected to GPIO10 (MOSI) - Pin 19
- MISO (Master In Slave Out) connected to GPIO9 (MISO) - Pin 21
- RST (Reset) connected to any GPIO (e.g., GPIO25) - Pin 22

#### RP4 - Additional buttons

- Play/Pause Button: Connected to GPIO17 - Pin 11
- Next Track Button: Connected to GPIO27 - Pin 13
- LED connected to GPIO22 - Pin 15
- All are connected to the GND - Pin 14

### RP0 Pinout

![RP0](docs/RP0-pinout.png)

#### Pi Zero - Connections for the SDA, SCK, MOSI, MISO, RST of the MFRC522

- SDA (Serial Data) connected to GPIO8 (CE0) - Pin 24
- SCK (Serial Clock) connected to GPIO11 (SCLK) - Pin 23
- MOSI (Master Out Slave In) connected to GPIO10 (MOSI) - Pin 19
- MISO (Master In Slave Out) connected to GPIO9 (MISO) - Pin 21
- RST (Reset) can be connected to any available GPIO. For consistency, you can still connect it to GPIO25, but ensure it doesn't interfere with other devices or functions.

#### Pi Zero - Additional buttons

- Play/Pause Button: Connected to GPIO17 - Pin 11
- Next Track Button: Connected to GPIO27 - Pin 13
- LED connected to GPIO22 - Pin 15
- All buttons and the LED should have their other side connected to GND - Pin 9 or 14 (or any other available GND pin).

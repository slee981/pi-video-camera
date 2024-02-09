# Pi Video Camera 

This provides proof of concept code to turn a Raspberry Pi with an add-on camera into a recording device that can use image recognition to trigger saving video segments to an Azure blob storage account. 

The idea is to explore the basics of how a video doorbell works by always watching (recording) and selectively saving. 

## Use 

1. Install Raspbian Lite OS onto a Raspberry Pi with camera installation.
2. Add required software using this bash script: https://github.com/slee981/raspi-opencv-setup.
3. Clone repo to Raspberry Pi. 
4. Setup an Azure Blob Storage account and set configuration in `main.go` storage.
5. Run:
    $ go run main.go

Of course, you can also compile and set to run on startup with appropriate sysctl configuration. 

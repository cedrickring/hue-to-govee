# Hue Govee Synchronizer

A Go application that synchronizes Philips Hue lights with Govee devices using the Hue v2 API and Govee LAN control features. This allows you to keep your Hue and Govee lights in sync, creating a unified lighting experience across different brands. 

## Prerequisites

- Philips Hue Bridge (v2 only)
- Govee devices with LAN control support
- Go 1.24+ (if building from source)
- Network access to both Hue Bridge and Govee devices (LAN control enabled)

## Configuration

Create a `config.yaml` file with your device settings:
```yaml
hue_bridge_id: "001788fffe123456"
hue_bridge_username: "abcdef1234567890abcdef1234567890abcdef12"

govee_multicast_ip: "239.255.255.250"

synchronizations:
- hue_light_id: "12345678-1234-5678-9abc-def012345678"
  hue_room_id: "87654321-4321-8765-cba9-876543210987"
  govee_device_id: "AA:BB:CC:DD:EE:FF:11:22" # living room lamp
- hue_light_id: "98765432-8765-4321-1234-567890abcdef"
  hue_room_id: "11223344-5566-7788-99aa-bbccddeeff00"
  govee_device_id: "11:22:33:44:55:66:77:88" # bedroom strip lights

log_level: "INFO"
```
### Configuration Parameters

- **hue_bridge_id**: Your Hue Bridge's unique identifier
- **hue_bridge_username**: Authentication username for API access
- **govee_multicast_ip**: Multicast IP for Govee device discovery (typically `239.255.255.250`)
- **synchronizations**: Array of light pairs to synchronize
  - **hue_light_id**: UUID of the Hue light device
  - **hue_room_id**: UUID of the Hue room containing the light
  - **govee_device_id**: MAC address of the Govee device
- **log_level**: Logging verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`)

Refer to the Philips Hue documentation on how to retrieve the bridge ID and username: https://developers.meethue.com/develop/get-started-2/

To get the Hue light and room ids, refer to the API documentation: https://developers.meethue.com/develop/hue-api-v2/

## Usage

### Docker
Use the following command to run `hue2govee` with docker:
```bash
docker run -d --rm -v $(pwd)/config.yaml:/config.yaml <image>
```

### Local

1. Create your `config.yaml` file with the appropriate values
2. Run the application:
   ```bash
   ./hue2govee
   ```

The application will start monitoring your configured Hue lights and synchronize their state (on/off, brightness, color) with the corresponding Govee devices.

## Troubleshooting

- **Bridge Connection Issues**: Ensure your bridge IP is correct and the bridge is on the same network
- **Authentication Errors**: Verify your username is valid and was created properly
- **Device Not Found**: Check that light IDs and room IDs are correct using the API endpoints above
- **Govee Connectivity**: Ensure Govee devices support LAN control and are on the same network
- **Network Issues**: Verify multicast traffic is allowed on your network for Govee discovery

## Development

### Building from Source

1. Clone the repository
2. Run `go build -o bin/hue2govee home/cmd/hue2govee` to build the application
3. Run `./hue2govee` to start the application

### Building Docker Image

1. Run `docker build -t <image> .` to build the image
2. Run `docker run -d --rm -v $(pwd)/config.yaml:/config.yaml <image>` to run the image

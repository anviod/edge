package server

import "github.com/gofiber/fiber/v2"

// scanDevice scans points in a device
func (s *Server) scanDevice(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	var params map[string]any
	if err := c.BodyParser(&params); err != nil {
		// Optional body
		params = make(map[string]any)
	}

	result, err := s.cm.ScanDevice(channelId, deviceId, params)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(result)
}

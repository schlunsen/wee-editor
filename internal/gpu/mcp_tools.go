package gpu

import (
	"encoding/json"
	"fmt"
)

// MCPContentBlock represents a content block in an MCP tool result.
type MCPContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPToolResult represents the result of an MCP tool call.
type MCPToolResult struct {
	Content []MCPContentBlock `json:"content"`
	IsError *bool             `json:"isError,omitempty"`
}

// ToolDefinition describes an MCP tool for registration in tools/list.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns MCP tool definitions for GPU operations.
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "gpu_launch",
			Description: "⚠️ BILLABLE REMOTE GPU — Launch a remote GPU pod on RunPod (costs $0.39–$3.49/hr). Only use when the user EXPLICITLY requests GPU compute for ML training, CUDA, or inference. This is NOT a local command — it provisions a remote server. For normal tasks, use Bash instead.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tier": map[string]interface{}{
						"type":        "string",
						"description": "GPU tier: starter (T4 16GB ~$0.39/hr), pro (A100 80GB ~$1.64/hr), or beast (H100 80GB ~$3.49/hr)",
						"enum":        []string{"starter", "pro", "beast"},
					},
					"template_id": map[string]interface{}{
						"type":        "string",
						"description": "Template ID for the GPU pod (use gpu_status to see available templates)",
					},
					"volume_gb": map[string]interface{}{
						"type":        "integer",
						"description": "Persistent volume size in GB. Tier defaults: starter=20, pro=50, beast=100. Values below tier default are raised automatically.",
					},
					"container_disk_gb": map[string]interface{}{
						"type":        "integer",
						"description": "Container disk size in GB. Tier defaults: starter=20, pro=40, beast=50. Values below tier default are raised automatically.",
					},
				},
				"required": []string{"tier", "template_id"},
			},
		},
		{
			Name:        "gpu_run",
			Description: "⚠️ REMOTE GPU EXECUTION (costs money) — Run a shell command on a remote RunPod GPU server via SSH. Only use for ML training, CUDA operations, or GPU-intensive tasks that require a GPU. For ALL normal commands (file operations, git, npm, builds, etc.), use the Bash tool instead — it runs locally for free.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Shell command to execute on the GPU pod",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "gpu_push",
			Description: "Sync files from the local sandbox to the remote GPU pod using rsync (or tar-over-SSH fallback). Excludes .git, node_modules, __pycache__ automatically. Only use when you need to transfer code/data to a running GPU pod for training or inference.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"local_path": map[string]interface{}{
						"type":        "string",
						"description": "Local path to sync (default: current working directory)",
					},
					"remote_path": map[string]interface{}{
						"type":        "string",
						"description": "Remote destination path on GPU pod (default: /workspace/project/)",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "gpu_pull",
			Description: "Sync files from the remote GPU pod back to the local sandbox using rsync. Only use to retrieve training results, model checkpoints, or outputs from a running GPU pod.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"remote_path": map[string]interface{}{
						"type":        "string",
						"description": "Remote path on GPU to sync (default: /app/)",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "gpu_status",
			Description: "Get status of the active remote GPU session including uptime, cost, and SSH connection details. Only relevant when a GPU pod has been launched.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_stop",
			Description: "Stop the active remote GPU pod (preserves volume, stops billing). Resume later with gpu_resume. Only use when a GPU pod is running.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_terminate",
			Description: "⚠️ DESTRUCTIVE — Terminate the active remote GPU pod permanently (deletes all data on the pod). Use this to clean up after ML training is done. IMPORTANT: always call this when finished with GPU work to avoid unnecessary billing.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_resume",
			Description: "⚠️ BILLABLE — Resume a previously stopped remote GPU pod (restores the same volume and data, resumes billing). Only use when the user explicitly asks to resume GPU work.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_templates",
			Description: "List available remote GPU pod templates (pre-configured container images with ML frameworks like PyTorch, TensorFlow, etc.). Only relevant when planning to launch a GPU pod.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_resize",
			Description: "Resize storage of the active remote GPU pod. Sizes can only increase. Volume changes apply immediately; container disk changes restart the pod. Only use when a GPU pod is running.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"volume_gb": map[string]interface{}{
						"type":        "integer",
						"description": "New persistent volume size in GB (must be >= current size)",
					},
					"container_disk_gb": map[string]interface{}{
						"type":        "integer",
						"description": "New container disk size in GB (must be >= current size, will restart pod)",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "gpu_list_volumes",
			Description: "List RunPod network volumes. Network volumes are persistent storage that can be mounted to different pods.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "gpu_create_volume",
			Description: "Create a new RunPod network volume for persistent storage across pods.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name for the network volume",
					},
					"size_gb": map[string]interface{}{
						"type":        "integer",
						"description": "Size of the volume in GB",
					},
					"data_center_id": map[string]interface{}{
						"type":        "string",
						"description": "Data center ID (e.g., 'US-TX-3'). Pod must be in same data center to mount.",
					},
				},
				"required": []string{"name", "size_gb", "data_center_id"},
			},
		},
		{
			Name:        "gpu_delete_volume",
			Description: "⚠️ DESTRUCTIVE & IRREVERSIBLE — Delete a RunPod network volume permanently. Only use when the user explicitly asks to delete a volume.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"volume_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the network volume to delete",
					},
				},
				"required": []string{"volume_id"},
			},
		},
	}
}

// HandleToolCall dispatches an MCP tool call to the appropriate GPU handler.
func HandleToolCall(manager *Manager, toolName string, args map[string]interface{}) MCPToolResult {
	switch toolName {
	case "gpu_launch":
		return handleGPULaunch(manager, args)
	case "gpu_run":
		return handleGPURun(manager, args)
	case "gpu_push":
		return handleGPUPush(manager, args)
	case "gpu_pull":
		return handleGPUPull(manager, args)
	case "gpu_status":
		return handleGPUStatus(manager, args)
	case "gpu_stop":
		return handleGPUStop(manager, args)
	case "gpu_terminate":
		return handleGPUTerminate(manager, args)
	case "gpu_resume":
		return handleGPUResume(manager, args)
	case "gpu_templates":
		return handleGPUTemplates(manager, args)
	case "gpu_resize":
		return handleGPUResize(manager, args)
	case "gpu_list_volumes":
		return handleGPUListVolumes(manager, args)
	case "gpu_create_volume":
		return handleGPUCreateVolume(manager, args)
	case "gpu_delete_volume":
		return handleGPUDeleteVolume(manager, args)
	default:
		return mcpError(fmt.Sprintf("Unknown GPU tool: %s", toolName))
	}
}

func mcpError(msg string) MCPToolResult {
	isError := true
	data, _ := json.Marshal(map[string]string{"error": msg})
	return MCPToolResult{
		Content: []MCPContentBlock{{Type: "text", Text: string(data)}},
		IsError: &isError,
	}
}

func mcpSuccess(result interface{}) MCPToolResult {
	data, _ := json.MarshalIndent(result, "", "  ")
	return MCPToolResult{
		Content: []MCPContentBlock{{Type: "text", Text: string(data)}},
	}
}

func handleGPULaunch(m *Manager, args map[string]interface{}) MCPToolResult {
	tier, _ := args["tier"].(string)
	if tier == "" {
		return mcpError("tier is required (starter, pro, or beast)")
	}
	templateID, _ := args["template_id"].(string)
	if templateID == "" {
		return mcpError("template_id is required")
	}

	pubKey, err := EnsureSSHKey(m.SSHKeyPath())
	if err != nil {
		return mcpError(fmt.Sprintf("SSH key error: %v", err))
	}

	volumeGB, _ := args["volume_gb"].(float64)
	containerDiskGB, _ := args["container_disk_gb"].(float64)

	session, err := m.Launch(tier, templateID, pubKey, int(volumeGB), int(containerDiskGB))
	if err != nil {
		return mcpError(fmt.Sprintf("Launch failed: %v", err))
	}

	result := map[string]interface{}{
		"success":  true,
		"pod_id":   session.PodID,
		"tier":     session.Tier,
		"gpu_type": session.GPUType,
		"status":   session.Status,
	}
	if session.SSHHost != "" {
		result["ssh_host"] = session.SSHHost
		result["ssh_port"] = session.SSHPort
	}
	if session.VolumeGB != nil {
		result["volume_gb"] = *session.VolumeGB
	}
	if session.ContainerDiskGB != nil {
		result["container_disk_gb"] = *session.ContainerDiskGB
	}
	return mcpSuccess(result)
}

func handleGPURun(m *Manager, args map[string]interface{}) MCPToolResult {
	command, _ := args["command"].(string)
	if command == "" {
		return mcpError("command is required")
	}

	output, err := m.RunCapture(command)
	if err != nil {
		// Still return partial output on error (useful for seeing why a command failed)
		if output != "" {
			return mcpError(fmt.Sprintf("Command failed: %v\n\nOutput:\n%s", err, output))
		}
		return mcpError(fmt.Sprintf("Command execution failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"command": command,
		"output":  output,
	})
}

func handleGPUPush(m *Manager, args map[string]interface{}) MCPToolResult {
	localPath, _ := args["local_path"].(string)
	remotePath, _ := args["remote_path"].(string)

	output, err := m.PushCapture(localPath, remotePath)
	if err != nil {
		if output != "" {
			return mcpError(fmt.Sprintf("Push failed: %v\n\nOutput:\n%s", err, output))
		}
		return mcpError(fmt.Sprintf("Push failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": output,
	})
}

func handleGPUPull(m *Manager, args map[string]interface{}) MCPToolResult {
	remotePath, _ := args["remote_path"].(string)

	output, err := m.PullCapture(remotePath)
	if err != nil {
		if output != "" {
			return mcpError(fmt.Sprintf("Pull failed: %v\n\nOutput:\n%s", err, output))
		}
		return mcpError(fmt.Sprintf("Pull failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": output,
	})
}

func handleGPUStatus(m *Manager, args map[string]interface{}) MCPToolResult {
	session, err := m.Status()
	if err != nil {
		// Try listing all sessions for more context
		sessions, listErr := m.ListSessions()
		if listErr != nil {
			return mcpError(fmt.Sprintf("No active GPU session: %v", err))
		}
		sessionList := make([]map[string]interface{}, 0, len(sessions))
		for _, s := range sessions {
			sessionList = append(sessionList, map[string]interface{}{
				"pod_id": s.PodID,
				"status": s.Status,
				"tier":   s.Tier,
			})
		}
		return mcpSuccess(map[string]interface{}{
			"active":   false,
			"message":  "No active GPU session",
			"sessions": sessionList,
		})
	}

	result := map[string]interface{}{
		"active":        true,
		"pod_id":        session.PodID,
		"tier":          session.Tier,
		"gpu_type":      session.GPUType,
		"status":        session.Status,
		"template_name": session.TemplateName,
	}
	if session.SSHHost != "" {
		result["ssh_host"] = session.SSHHost
		result["ssh_port"] = session.SSHPort
	}
	if session.VolumeGB != nil {
		result["volume_gb"] = *session.VolumeGB
	}
	if session.ContainerDiskGB != nil {
		result["container_disk_gb"] = *session.ContainerDiskGB
	}
	if session.Live != nil {
		result["cost_per_hr"] = session.Live.CostPerHr
		result["uptime_seconds"] = session.Live.UptimeSeconds
		result["total_cost"] = session.Live.TotalCost
	}
	return mcpSuccess(result)
}

func handleGPUTemplates(m *Manager, args map[string]interface{}) MCPToolResult {
	templates, err := m.ListTemplates()
	if err != nil {
		return mcpError(fmt.Sprintf("Failed to list templates: %v", err))
	}

	templateList := make([]map[string]interface{}, 0, len(templates))
	for _, t := range templates {
		templateList = append(templateList, map[string]interface{}{
			"id":         t.ID,
			"name":       t.Name,
			"image_name": t.ImageName,
		})
	}

	return mcpSuccess(map[string]interface{}{
		"templates": templateList,
		"tiers": map[string]interface{}{
			"starter": "RTX 4090 (~$0.39/hr) - 20GB volume + 20GB disk default",
			"pro":     "A100 80GB (~$1.64/hr) - 50GB volume + 40GB disk default",
			"beast":   "H100 80GB (~$3.49/hr) - 100GB volume + 50GB disk default",
		},
	})
}

func handleGPUStop(m *Manager, args map[string]interface{}) MCPToolResult {
	err := m.Stop()
	if err != nil {
		return mcpError(fmt.Sprintf("Stop failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": "GPU pod stopped (volume preserved). Resume with gpu_resume.",
	})
}

func handleGPUTerminate(m *Manager, args map[string]interface{}) MCPToolResult {
	err := m.Kill()
	if err != nil {
		return mcpError(fmt.Sprintf("Terminate failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": "GPU pod terminated permanently. All pod data has been deleted.",
	})
}

func handleGPUResume(m *Manager, args map[string]interface{}) MCPToolResult {
	err := m.Resume()
	if err != nil {
		return mcpError(fmt.Sprintf("Resume failed: %v", err))
	}

	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": "GPU pod resuming. Check gpu_status for progress.",
	})
}

func handleGPUResize(m *Manager, args map[string]interface{}) MCPToolResult {
	active, err := m.ActiveSession()
	if err != nil || active == nil {
		return mcpError("No active GPU session to resize")
	}
	volumeGB, _ := args["volume_gb"].(float64)
	containerDiskGB, _ := args["container_disk_gb"].(float64)
	if volumeGB == 0 && containerDiskGB == 0 {
		return mcpError("At least one of volume_gb or container_disk_gb must be specified")
	}
	err = m.ResizePod(active.ID, int(volumeGB), int(containerDiskGB))
	if err != nil {
		return mcpError(fmt.Sprintf("Resize failed: %v", err))
	}
	result := map[string]interface{}{
		"success": true,
		"message": "Storage resized successfully",
	}
	if volumeGB > 0 {
		result["volume_gb"] = int(volumeGB)
	}
	if containerDiskGB > 0 {
		result["container_disk_gb"] = int(containerDiskGB)
		result["note"] = "Container disk change will restart the pod"
	}
	return mcpSuccess(result)
}

func handleGPUListVolumes(m *Manager, args map[string]interface{}) MCPToolResult {
	volumes, err := m.ListNetworkVolumes()
	if err != nil {
		return mcpError(fmt.Sprintf("Failed to list volumes: %v", err))
	}
	volumeList := make([]map[string]interface{}, 0, len(volumes))
	for _, v := range volumes {
		vol := map[string]interface{}{
			"id":             v.ID,
			"name":           v.Name,
			"size_gb":        v.SizeGB,
			"data_center_id": v.DataCenterID,
		}
		if v.MountedPodID != "" {
			vol["mounted_pod_id"] = v.MountedPodID
		}
		volumeList = append(volumeList, vol)
	}
	return mcpSuccess(map[string]interface{}{
		"volumes": volumeList,
	})
}

func handleGPUCreateVolume(m *Manager, args map[string]interface{}) MCPToolResult {
	name, _ := args["name"].(string)
	if name == "" {
		return mcpError("name is required")
	}
	sizeGB, _ := args["size_gb"].(float64)
	if sizeGB <= 0 {
		return mcpError("size_gb must be > 0")
	}
	dataCenterID, _ := args["data_center_id"].(string)
	if dataCenterID == "" {
		return mcpError("data_center_id is required")
	}
	vol, err := m.CreateNetworkVolume(name, int(sizeGB), dataCenterID)
	if err != nil {
		return mcpError(fmt.Sprintf("Failed to create volume: %v", err))
	}
	return mcpSuccess(map[string]interface{}{
		"success":        true,
		"id":             vol.ID,
		"name":           vol.Name,
		"size_gb":        vol.SizeGB,
		"data_center_id": vol.DataCenterID,
	})
}

func handleGPUDeleteVolume(m *Manager, args map[string]interface{}) MCPToolResult {
	volumeID, _ := args["volume_id"].(string)
	if volumeID == "" {
		return mcpError("volume_id is required")
	}
	err := m.DeleteNetworkVolume(volumeID)
	if err != nil {
		return mcpError(fmt.Sprintf("Failed to delete volume: %v", err))
	}
	return mcpSuccess(map[string]interface{}{
		"success": true,
		"message": "Network volume deleted permanently",
	})
}

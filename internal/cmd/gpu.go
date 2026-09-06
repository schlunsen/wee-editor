package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/schlunsen/wee-editor/internal/gpu"
	"github.com/spf13/cobra"
)

var (
	gpuTier            string
	gpuTemplate        string
	gpuPath            string
	gpuVolumeGB        int
	gpuContainerDiskGB int
)

var gpuCmd = &cobra.Command{
	Use:   "gpu",
	Short: "Manage GPU sidecar for heavy compute workloads",
	Long: `Manage GPU sidecar sessions powered by RunPod.

Launch on-demand GPU pods from your sandbox for ML training, fine-tuning,
inference, and other GPU-intensive tasks. Files can be synced between
the sandbox and GPU pod via push/pull.

SETUP (one-time):
  Connect your RunPod API key in the wee.cat dashboard under
  Settings → GPU Providers. Get a key at https://runpod.io/console/user/settings

WORKFLOW:
  1. wee gpu templates              # List available pod templates
  2. wee gpu launch --tier pro \    # Launch a GPU pod
       --template <template_id>
  3. wee gpu status                 # Wait for "running" + SSH ready
  4. wee gpu push                   # Sync your code to the GPU
  5. wee gpu run "python train.py"  # Run training on the GPU
  6. wee gpu pull                   # Pull results back to sandbox
  7. wee gpu kill                   # Terminate pod (stops billing)

TIERS:
  starter   RTX 4090 24GB     ~$0.39/hr   Small models (<3B params)
  pro       A100 80GB         ~$1.64/hr   Medium models (3-13B)
  beast     H100 80GB HBM3    ~$3.49/hr   Large models (13B+)

  If the requested tier is sold out, the system automatically tries
  the next tier up (starter → pro → beast).

EXAMPLES:
  wee gpu                                  # Interactive TUI picker
  wee gpu launch --tier starter \
    --template abc123                      # Launch starter GPU
  wee gpu run "nvidia-smi"                 # Check GPU info
  wee gpu run "pip install unsloth && \
    python train.py"                       # Install deps + train
  wee gpu shell                            # Interactive SSH session
  wee gpu push ./my-project                # Sync specific directory
  wee gpu pull /app/output                 # Pull results from GPU
  wee gpu stop                             # Pause (keep volume)
  wee gpu resume                           # Resume paused pod
  wee gpu kill -y                          # Terminate without prompt

MCP TOOLS (for AI agents):
  When running inside a wee.cat sandbox, these are available as MCP
  tools for Claude Code or other agents:
    gpu_templates, gpu_launch, gpu_status, gpu_run,
    gpu_push, gpu_pull, gpu_stop, gpu_resume, gpu_terminate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If tier and template are both set, fall through to launch
		if gpuTier != "" && gpuTemplate != "" {
			return gpuLaunchCmd.RunE(cmd, args)
		}

		// Otherwise, launch TUI
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}
		return gpu.RunTUI(mgr)
	},
}

var gpuLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch a new GPU pod",
	Long: `Launch a new GPU pod with the specified tier and template.

If --tier or --template is omitted, an interactive TUI will guide you
through selection. Use 'wee gpu templates' to list available templates.

The pod launches asynchronously — use 'wee gpu status' to track progress.
SSH connection details appear once the pod reaches "running" state.

If the requested GPU tier is sold out, the system automatically tries
the next tier up (starter → pro → beast) so your launch succeeds.`,
	Example: `  wee gpu launch --tier starter --template abc123
  wee gpu launch --tier pro --template abc123
  wee gpu launch   # interactive mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		// If tier or template missing, launch TUI
		if gpuTier == "" || gpuTemplate == "" {
			return gpu.RunTUI(mgr)
		}

		spinner := ShowSpinner("Generating SSH key...")
		pubKey, err := gpu.EnsureSSHKey(mgr.SSHKeyPath())
		if err != nil {
			spinner.Fail("Failed to generate SSH key")
			return fmt.Errorf("SSH key error: %w", err)
		}
		spinner.Success("SSH key ready")

		spinner = ShowSpinner(fmt.Sprintf("Launching %s GPU pod...", gpuTier))
		session, err := mgr.Launch(gpuTier, gpuTemplate, pubKey, gpuVolumeGB, gpuContainerDiskGB)
		if err != nil {
			spinner.Fail("Failed to launch GPU pod")
			return fmt.Errorf("launch failed: %w", err)
		}
		spinner.Success("GPU pod launched!")

		content := fmt.Sprintf(
			"Pod ID:   %s\nTier:     %s\nGPU:      %s\nStatus:   %s",
			session.PodID, session.Tier, session.GPUType, session.Status,
		)
		if session.VolumeGB != nil {
			content += fmt.Sprintf("\nVolume:   %d GB", *session.VolumeGB)
		}
		if session.ContainerDiskGB != nil {
			content += fmt.Sprintf("\nDisk:     %d GB", *session.ContainerDiskGB)
		}
		ShowBox("GPU Session", content)

		ShowInfo("Next steps:")
		ShowInfo("  wee gpu status   - check progress")
		ShowInfo("  wee gpu shell    - connect via SSH")
		ShowInfo("  wee gpu push     - sync files to GPU")
		ShowInfo("  wee gpu kill     - terminate when done")

		return nil
	},
}

var gpuTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available GPU pod templates",
	Long: `List RunPod templates available for launching GPU pods.

Templates are pre-configured container images with ML frameworks
(PyTorch, TensorFlow, etc.) already installed. Use the template ID
with 'wee gpu launch --template <id>'.

Templates are configured in your RunPod account at:
  https://runpod.io/console/user/templates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		spinner := ShowSpinner("Fetching templates...")
		templates, err := mgr.ListTemplates()
		if err != nil {
			spinner.Fail("Failed to fetch templates")
			return err
		}
		spinner.Success(fmt.Sprintf("Found %d templates", len(templates)))

		if len(templates) == 0 {
			ShowWarning("No templates found. Create one at https://runpod.io/console/user/templates")
			return nil
		}

		for _, t := range templates {
			fmt.Printf("  %-25s  %s  [%s]\n", t.Name, t.ID, t.ImageName)
		}

		ShowInfo("\nUse a template ID with: wee gpu launch --tier <tier> --template <id>")
		return nil
	},
}

var gpuStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GPU session status, SSH details, and cost",
	Long: `Show the current GPU session status with live data from RunPod.

Displays pod ID, tier, GPU type, status, SSH connection details,
uptime, and estimated cost. SSH host/port appear once the pod
finishes booting (status changes from "creating" to "running").`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		spinner := ShowSpinner("Fetching GPU status...")
		session, err := mgr.Status()
		if err != nil {
			spinner.Fail("No active GPU session")
			return err
		}
		spinner.Success("GPU session active")

		content := fmt.Sprintf(
			"Pod ID:   %s\nTier:     %s\nGPU:      %s\nStatus:   %s\nTemplate: %s",
			session.PodID, session.Tier, session.GPUType, session.Status, session.TemplateName,
		)

		if session.VolumeGB != nil {
			content += fmt.Sprintf("\nVolume:   %d GB", *session.VolumeGB)
		}
		if session.ContainerDiskGB != nil {
			content += fmt.Sprintf("\nDisk:     %d GB", *session.ContainerDiskGB)
		}

		if session.ErrorMessage != "" {
			content += fmt.Sprintf("\n\nError:    %s", session.ErrorMessage)
		}

		if session.Live != nil {
			uptimeMin := session.Live.UptimeSeconds / 60
			content += fmt.Sprintf(
				"\n\nCost/hr:  $%.2f\nUptime:   %d min\nTotal:    $%.4f",
				session.Live.CostPerHr, uptimeMin, session.Live.TotalCost,
			)
		}

		if session.SSHHost != "" {
			content += fmt.Sprintf("\n\nSSH:      root@%s -p %d", session.SSHHost, session.SSHPort)
		}

		ShowBox("GPU Status", content)
		return nil
	},
}

var gpuShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive SSH session to the GPU pod",
	Long: `Open an interactive shell on the active GPU pod.

Uses Go's native SSH library — no ssh binary required. Supports
full PTY with color, tab completion, and window resizing.

The pod must be in "running" state with SSH details available.
If you just launched, wait a moment and check 'wee gpu status'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		ShowInfo("Connecting to GPU pod via SSH...")
		return mgr.Shell()
	},
}

var gpuRunCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run a command on the GPU pod via SSH",
	Long: `Execute a command on the active GPU pod and stream output.

The command runs as root on the GPU pod. Stdout and stderr are
streamed back in real-time. Use quotes for multi-word commands.`,
	Example: `  wee gpu run "nvidia-smi"
  wee gpu run "python train.py"
  wee gpu run "pip install torch && python -c 'import torch; print(torch.cuda.is_available())'"
  wee gpu run "ls -la /app/"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		remoteCmd := strings.Join(args, " ")
		ShowInfo(fmt.Sprintf("Running on GPU: %s", remoteCmd))
		return mgr.Run(remoteCmd)
	},
}

var gpuPushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Sync files from sandbox to GPU pod",
	Long: `Sync local files to the GPU pod using rsync over SSH.

Defaults to syncing the current working directory to /app/ on the GPU.
Use this to push your training scripts, datasets, and configs before
running training on the GPU.`,
	Example: `  wee gpu push                  # sync current dir to /app/
  wee gpu push ./training       # sync specific directory
  wee gpu push --path ./data    # alternative syntax`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		path := gpuPath
		if path == "" && len(args) > 0 {
			path = args[0]
		}

		spinner := ShowSpinner("Syncing files to GPU...")
		if err := mgr.Push(path); err != nil {
			spinner.Fail("Push failed")
			return err
		}
		spinner.Success("Files synced to GPU")
		return nil
	},
}

var gpuPullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Sync files from GPU pod to sandbox",
	Long: `Sync files from the GPU pod to the local sandbox using rsync over SSH.

Defaults to pulling from /app/ on the GPU to the current directory.
Use this to retrieve trained models, checkpoints, and outputs after
training completes.`,
	Example: `  wee gpu pull                     # pull /app/ to current dir
  wee gpu pull /app/output         # pull specific remote dir
  wee gpu pull --path /app/models  # alternative syntax`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		path := gpuPath
		if path == "" && len(args) > 0 {
			path = args[0]
		}

		spinner := ShowSpinner("Syncing files from GPU...")
		if err := mgr.Pull(path); err != nil {
			spinner.Fail("Pull failed")
			return err
		}
		spinner.Success("Files synced from GPU")
		return nil
	},
}

var gpuStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the GPU pod (preserves volume, stops billing)",
	Long: `Stop the active GPU pod. The volume is preserved so you can
resume later without losing data. Billing stops while the pod is stopped.

Use 'wee gpu resume' to restart the pod with the same volume.
Use 'wee gpu kill' to permanently terminate and delete all data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		spinner := ShowSpinner("Stopping GPU pod...")
		if err := mgr.Stop(); err != nil {
			spinner.Fail("Failed to stop GPU pod")
			return err
		}
		spinner.Success("GPU pod stopped (volume preserved)")
		ShowInfo("Resume later with: wee gpu resume")
		return nil
	},
}

var gpuResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a previously stopped GPU pod",
	Long: `Resume a GPU pod that was previously stopped with 'wee gpu stop'.

The pod restarts with the same volume and data intact. Use
'wee gpu status' to track when it becomes "running" again.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		spinner := ShowSpinner("Resuming GPU pod...")
		if err := mgr.Resume(); err != nil {
			spinner.Fail("Failed to resume GPU pod")
			return err
		}
		spinner.Success("GPU pod resumed!")
		ShowInfo("Check status with: wee gpu status")
		return nil
	},
}

var gpuKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Terminate GPU pod permanently (deletes all data)",
	Long: `Permanently terminate the active GPU pod and delete all data.

This is irreversible — all files on the pod (including the volume)
are deleted. Make sure to 'wee gpu pull' any results you need first.

IMPORTANT: Always kill your GPU pod when done to stop billing.
Use -y to skip the confirmation prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getGPUManager()
		if err != nil {
			return err
		}

		// Confirmation prompt
		if !yesFlag {
			ShowWarning("This will permanently terminate the GPU pod and delete all data on it.")
			fmt.Print("Are you sure? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				ShowInfo("Cancelled.")
				return nil
			}
		}

		spinner := ShowSpinner("Terminating GPU pod...")
		if err := mgr.Kill(); err != nil {
			spinner.Fail("Failed to terminate GPU pod")
			return err
		}
		spinner.Success("GPU pod terminated")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gpuCmd)
	gpuCmd.AddCommand(gpuLaunchCmd, gpuTemplatesCmd, gpuStatusCmd, gpuShellCmd, gpuRunCmd, gpuPushCmd, gpuPullCmd, gpuStopCmd, gpuResumeCmd, gpuKillCmd)

	// Flags on root gpu command (for shortcut launch)
	gpuCmd.Flags().StringVar(&gpuTier, "tier", "", "GPU tier: starter, pro, or beast")
	gpuCmd.Flags().StringVar(&gpuTemplate, "template", "", "Template ID (see: wee gpu templates)")
	gpuCmd.Flags().IntVar(&gpuVolumeGB, "volume", 0, "Persistent volume size in GB (0 = tier default)")
	gpuCmd.Flags().IntVar(&gpuContainerDiskGB, "disk", 0, "Container disk size in GB (0 = tier default)")

	// Flags on launch subcommand
	gpuLaunchCmd.Flags().StringVar(&gpuTier, "tier", "", "GPU tier: starter, pro, or beast")
	gpuLaunchCmd.Flags().StringVar(&gpuTemplate, "template", "", "Template ID (see: wee gpu templates)")
	gpuLaunchCmd.Flags().IntVar(&gpuVolumeGB, "volume", 0, "Persistent volume size in GB (0 = tier default)")
	gpuLaunchCmd.Flags().IntVar(&gpuContainerDiskGB, "disk", 0, "Container disk size in GB (0 = tier default)")

	// Flags on push/pull
	gpuPushCmd.Flags().StringVar(&gpuPath, "path", "", "Local path to sync (default: current directory)")
	gpuPullCmd.Flags().StringVar(&gpuPath, "path", "", "Remote path to sync (default: /app/)")
}

func getGPUManager() (*gpu.Manager, error) {
	mgr, err := gpu.NewManagerFromEnv()
	if err != nil {
		ShowError("GPU sidecar requires a wee.cat sandbox environment")
		ShowInfo("Required environment variables:")
		ShowInfo("  WEE_CONTROL_PLANE_URL  - Control plane API URL")
		ShowInfo("  WEE_SANDBOX_ID         - Sandbox identifier")
		ShowInfo("  WEE_CONTROL_PLANE_TOKEN - Authentication token")
		return nil, fmt.Errorf("GPU sidecar requires a wee.cat sandbox environment: %w", err)
	}
	return mgr, nil
}

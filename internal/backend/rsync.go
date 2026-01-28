package backend

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// RsyncOptions holds parsed rsync flags
type RsyncOptions struct {
	Archive   bool
	Verbose   bool
	Compress  bool
	Delete    bool
	DryRun    bool
	Recursive bool
	Exclude   []string
	Extra     []string // passthrough for SSH, warned+ignored for Docker/K8s
}

// Rsyncer is an optional interface backends can implement
// to support rsync-style file synchronization
type Rsyncer interface {
	Rsync(ctx context.Context, host *Host, localPath, remotePath, direction string, opts *RsyncOptions) error
}

// Rsync dispatches to the backend's Rsync method if supported
func Rsync(ctx context.Context, b Backend, host *Host, localPath, remotePath, direction string, opts *RsyncOptions) error {
	r, ok := b.(Rsyncer)
	if !ok {
		return fmt.Errorf("backend %q does not support rsync", b.Name())
	}
	return r.Rsync(ctx, host, localPath, remotePath, direction, opts)
}

// buildRsyncFlags converts RsyncOptions to CLI flags for native rsync
func buildRsyncFlags(opts *RsyncOptions) []string {
	var flags []string

	if opts.Archive {
		flags = append(flags, "-a")
	}
	if opts.Verbose {
		flags = append(flags, "-v")
	}
	if opts.Compress {
		flags = append(flags, "-z")
	}
	if opts.Recursive {
		flags = append(flags, "-r")
	}
	if opts.DryRun {
		flags = append(flags, "-n")
	}
	if opts.Delete {
		flags = append(flags, "--delete")
	}
	for _, pattern := range opts.Exclude {
		flags = append(flags, "--exclude="+pattern)
	}
	flags = append(flags, opts.Extra...)

	return flags
}

// shellQuote quotes a string for safe use in shell commands
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// SSH backend rsync implementation

func (s *SSHBackend) Rsync(ctx context.Context, host *Host, localPath, remotePath, direction string, opts *RsyncOptions) error {
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync is not installed locally")
	}

	// Build -e flag for SSH transport
	sshCmd := "ssh"
	if port := host.Properties["port"]; port != "" && port != "22" {
		sshCmd += " -p " + port
	}
	if identity := host.Properties["identity"]; identity != "" {
		sshCmd += " -i " + shellQuote(identity)
	}
	if jump := host.Properties["jump"]; jump != "" {
		sshCmd += " -J " + jump
	}

	var args []string
	args = append(args, buildRsyncFlags(opts)...)
	args = append(args, "-e", sshCmd)

	dest := s.buildDestination(host)
	if direction == "to" {
		args = append(args, localPath, fmt.Sprintf("%s:%s", dest, remotePath))
	} else {
		args = append(args, fmt.Sprintf("%s:%s", dest, remotePath), localPath)
	}

	cmd := exec.CommandContext(ctx, "rsync", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Docker backend rsync implementation (tar-streaming fallback)

func (d *DockerBackend) Rsync(ctx context.Context, host *Host, localPath, remotePath, direction string, opts *RsyncOptions) error {
	container, err := d.resolveContainer(ctx, host)
	if err != nil {
		return err
	}

	if len(opts.Extra) > 0 {
		fmt.Fprintf(os.Stderr, "warning: unsupported rsync flags ignored for docker backend: %s\n", strings.Join(opts.Extra, " "))
	}

	if opts.DryRun {
		if direction == "to" {
			fmt.Printf("[dry-run] would sync %s -> %s:%s\n", localPath, container, remotePath)
		} else {
			fmt.Printf("[dry-run] would sync %s:%s -> %s\n", container, remotePath, localPath)
		}
		if opts.Delete {
			fmt.Println("[dry-run] would delete extraneous files at destination")
		}
		return nil
	}

	if direction == "to" {
		return d.rsyncTo(ctx, container, localPath, remotePath, opts)
	}
	return d.rsyncFrom(ctx, container, localPath, remotePath, opts)
}

func (d *DockerBackend) rsyncTo(ctx context.Context, container, localPath, remotePath string, opts *RsyncOptions) error {
	// Validate local path exists
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("local path %q: %w", localPath, err)
	}

	// Delete destination contents first if requested
	if opts.Delete {
		delCmd := exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", fmt.Sprintf("rm -rf %s/*", shellQuote(remotePath)))
		delCmd.Stderr = os.Stderr
		if err := delCmd.Run(); err != nil {
			return fmt.Errorf("failed to clean remote directory: %w", err)
		}
	}

	// Ensure destination directory exists
	mkdirCmd := exec.CommandContext(ctx, "docker", "exec", container, "mkdir", "-p", remotePath)
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Build tar create command
	tarArgs := []string{"-cf", "-"}
	for _, pattern := range opts.Exclude {
		tarArgs = append(tarArgs, "--exclude="+pattern)
	}
	if opts.Compress {
		tarArgs = append(tarArgs, "-z")
	}
	tarArgs = append(tarArgs, "-C", localPath, ".")

	tarCreate := exec.CommandContext(ctx, "tar", tarArgs...)
	tarCreate.Stderr = os.Stderr

	// Build docker exec tar extract command
	dockerExtractArgs := []string{"exec", "-i", container, "tar", "-xf", "-"}
	if opts.Compress {
		dockerExtractArgs = append(dockerExtractArgs, "-z")
	}
	dockerExtractArgs = append(dockerExtractArgs, "-C", remotePath)

	dockerExtract := exec.CommandContext(ctx, "docker", dockerExtractArgs...)
	dockerExtract.Stderr = os.Stderr
	if opts.Verbose {
		dockerExtract.Stdout = os.Stdout
	}

	// Pipe tar output to docker extract
	pipe, err := tarCreate.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	dockerExtract.Stdin = pipe

	if err := tarCreate.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}
	if err := dockerExtract.Start(); err != nil {
		tarCreate.Process.Kill()
		return fmt.Errorf("failed to start docker exec: %w", err)
	}

	tarErr := tarCreate.Wait()
	dockerErr := dockerExtract.Wait()

	if tarErr != nil {
		return fmt.Errorf("tar create failed: %w", tarErr)
	}
	if dockerErr != nil {
		return fmt.Errorf("docker extract failed: %w", dockerErr)
	}

	return nil
}

func (d *DockerBackend) rsyncFrom(ctx context.Context, container, localPath, remotePath string, opts *RsyncOptions) error {
	// Delete local contents first if requested
	if opts.Delete {
		entries, err := os.ReadDir(localPath)
		if err == nil {
			for _, entry := range entries {
				os.RemoveAll(localPath + "/" + entry.Name())
			}
		}
	}

	// Ensure local directory exists
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Build docker exec tar create command
	dockerCreateArgs := []string{"exec", container, "tar", "-cf", "-"}
	for _, pattern := range opts.Exclude {
		dockerCreateArgs = append(dockerCreateArgs, "--exclude="+pattern)
	}
	if opts.Compress {
		dockerCreateArgs = append(dockerCreateArgs, "-z")
	}
	dockerCreateArgs = append(dockerCreateArgs, "-C", remotePath, ".")

	dockerCreate := exec.CommandContext(ctx, "docker", dockerCreateArgs...)
	dockerCreate.Stderr = os.Stderr

	// Build local tar extract command
	tarExtractArgs := []string{"-xf", "-"}
	if opts.Compress {
		tarExtractArgs = append(tarExtractArgs, "-z")
	}
	tarExtractArgs = append(tarExtractArgs, "-C", localPath)

	tarExtract := exec.CommandContext(ctx, "tar", tarExtractArgs...)
	tarExtract.Stderr = os.Stderr
	if opts.Verbose {
		tarExtract.Stdout = os.Stdout
	}

	// Pipe docker tar output to local tar extract
	pipe, err := dockerCreate.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	tarExtract.Stdin = pipe

	if err := dockerCreate.Start(); err != nil {
		return fmt.Errorf("failed to start docker exec: %w", err)
	}
	if err := tarExtract.Start(); err != nil {
		dockerCreate.Process.Kill()
		return fmt.Errorf("failed to start tar extract: %w", err)
	}

	dockerErr := dockerCreate.Wait()
	tarErr := tarExtract.Wait()

	if dockerErr != nil {
		return fmt.Errorf("docker tar create failed: %w", dockerErr)
	}
	if tarErr != nil {
		return fmt.Errorf("tar extract failed: %w", tarErr)
	}

	return nil
}

// K8s backend rsync implementation (tar-streaming fallback)

func (k *K8sBackend) Rsync(ctx context.Context, host *Host, localPath, remotePath, direction string, opts *RsyncOptions) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	if len(opts.Extra) > 0 {
		fmt.Fprintf(os.Stderr, "warning: unsupported rsync flags ignored for k8s backend: %s\n", strings.Join(opts.Extra, " "))
	}

	if opts.DryRun {
		if direction == "to" {
			fmt.Printf("[dry-run] would sync %s -> %s:%s\n", localPath, pod, remotePath)
		} else {
			fmt.Printf("[dry-run] would sync %s:%s -> %s\n", pod, remotePath, localPath)
		}
		if opts.Delete {
			fmt.Println("[dry-run] would delete extraneous files at destination")
		}
		return nil
	}

	if direction == "to" {
		return k.rsyncTo(ctx, host, pod, localPath, remotePath, opts)
	}
	return k.rsyncFrom(ctx, host, pod, localPath, remotePath, opts)
}

func (k *K8sBackend) buildExecArgs(host *Host, pod string, command ...string) []string {
	args := k.buildBaseArgs(host)
	args = append(args, "exec", pod)
	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--")
	args = append(args, command...)
	return args
}

func (k *K8sBackend) buildExecArgsStdin(host *Host, pod string, command ...string) []string {
	args := k.buildBaseArgs(host)
	args = append(args, "exec", "-i", pod)
	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--")
	args = append(args, command...)
	return args
}

func (k *K8sBackend) rsyncTo(ctx context.Context, host *Host, pod, localPath, remotePath string, opts *RsyncOptions) error {
	// Validate local path exists
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("local path %q: %w", localPath, err)
	}

	// Delete destination contents first if requested
	if opts.Delete {
		delArgs := k.buildExecArgs(host, pod, "sh", "-c", fmt.Sprintf("rm -rf %s/*", shellQuote(remotePath)))
		delCmd := exec.CommandContext(ctx, "kubectl", delArgs...)
		delCmd.Stderr = os.Stderr
		if err := delCmd.Run(); err != nil {
			return fmt.Errorf("failed to clean remote directory: %w", err)
		}
	}

	// Ensure destination directory exists
	mkdirArgs := k.buildExecArgs(host, pod, "mkdir", "-p", remotePath)
	mkdirCmd := exec.CommandContext(ctx, "kubectl", mkdirArgs...)
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Build tar create command
	tarArgs := []string{"-cf", "-"}
	for _, pattern := range opts.Exclude {
		tarArgs = append(tarArgs, "--exclude="+pattern)
	}
	if opts.Compress {
		tarArgs = append(tarArgs, "-z")
	}
	tarArgs = append(tarArgs, "-C", localPath, ".")

	tarCreate := exec.CommandContext(ctx, "tar", tarArgs...)
	tarCreate.Stderr = os.Stderr

	// Build kubectl exec tar extract command
	extractTarArgs := []string{"-xf", "-"}
	if opts.Compress {
		extractTarArgs = append(extractTarArgs, "-z")
	}
	extractTarArgs = append(extractTarArgs, "-C", remotePath)
	kubectlArgs := k.buildExecArgsStdin(host, pod, append([]string{"tar"}, extractTarArgs...)...)

	kubectlExtract := exec.CommandContext(ctx, "kubectl", kubectlArgs...)
	kubectlExtract.Stderr = os.Stderr
	if opts.Verbose {
		kubectlExtract.Stdout = os.Stdout
	}

	// Pipe tar output to kubectl extract
	pipe, err := tarCreate.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	kubectlExtract.Stdin = pipe

	if err := tarCreate.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}
	if err := kubectlExtract.Start(); err != nil {
		tarCreate.Process.Kill()
		return fmt.Errorf("failed to start kubectl exec: %w", err)
	}

	tarErr := tarCreate.Wait()
	kubectlErr := kubectlExtract.Wait()

	if tarErr != nil {
		return fmt.Errorf("tar create failed: %w", tarErr)
	}
	if kubectlErr != nil {
		return fmt.Errorf("kubectl extract failed: %w", kubectlErr)
	}

	return nil
}

func (k *K8sBackend) rsyncFrom(ctx context.Context, host *Host, pod, localPath, remotePath string, opts *RsyncOptions) error {
	// Delete local contents first if requested
	if opts.Delete {
		entries, err := os.ReadDir(localPath)
		if err == nil {
			for _, entry := range entries {
				os.RemoveAll(localPath + "/" + entry.Name())
			}
		}
	}

	// Ensure local directory exists
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Build kubectl exec tar create command
	createTarArgs := []string{"-cf", "-"}
	for _, pattern := range opts.Exclude {
		createTarArgs = append(createTarArgs, "--exclude="+pattern)
	}
	if opts.Compress {
		createTarArgs = append(createTarArgs, "-z")
	}
	createTarArgs = append(createTarArgs, "-C", remotePath, ".")
	kubectlArgs := k.buildExecArgs(host, pod, append([]string{"tar"}, createTarArgs...)...)

	kubectlCreate := exec.CommandContext(ctx, "kubectl", kubectlArgs...)
	kubectlCreate.Stderr = os.Stderr

	// Build local tar extract command
	tarExtractArgs := []string{"-xf", "-"}
	if opts.Compress {
		tarExtractArgs = append(tarExtractArgs, "-z")
	}
	tarExtractArgs = append(tarExtractArgs, "-C", localPath)

	tarExtract := exec.CommandContext(ctx, "tar", tarExtractArgs...)
	tarExtract.Stderr = os.Stderr
	if opts.Verbose {
		tarExtract.Stdout = os.Stdout
	}

	// Pipe kubectl tar output to local tar extract
	pr, pw := io.Pipe()
	kubectlCreate.Stdout = pw
	tarExtract.Stdin = pr

	if err := kubectlCreate.Start(); err != nil {
		return fmt.Errorf("failed to start kubectl exec: %w", err)
	}
	if err := tarExtract.Start(); err != nil {
		kubectlCreate.Process.Kill()
		return fmt.Errorf("failed to start tar extract: %w", err)
	}

	// Close the write end when kubectl finishes so tar sees EOF
	go func() {
		kubectlCreate.Wait()
		pw.Close()
	}()

	tarErr := tarExtract.Wait()
	kubectlErr := kubectlCreate.Wait()

	if kubectlErr != nil {
		return fmt.Errorf("kubectl tar create failed: %w", kubectlErr)
	}
	if tarErr != nil {
		return fmt.Errorf("tar extract failed: %w", tarErr)
	}

	return nil
}

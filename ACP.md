# Agent Client Protocol (ACP) Integration

Crush now supports the [Agent Client Protocol (ACP)](https://agentclientprotocol.com/), enabling seamless integration with ACP-compatible IDEs and editors.

## What is ACP?

The Agent Client Protocol standardizes communication between code editors and AI coding agents using JSON-RPC 2.0. This allows any ACP-compatible editor to work with any ACP-compatible agent, including Crush.

## Using Crush as an ACP Agent

### Basic Usage

Run Crush in ACP mode:

```bash
crush acp
```

This starts Crush as an ACP agent that communicates via JSON-RPC 2.0 over stdin/stdout.

### With Debug Logging

To see detailed logs (sent to stderr):

```bash
crush acp -d
```

Logs are also saved to `./.crush/logs/crush.log` relative to your project directory.

## Editor Integration

### Zed Editor

[Zed](https://zed.dev/) has native support for ACP agents. To configure Crush in Zed:

1. Open Zed's settings (Cmd/Ctrl + ,)
2. Add Crush to the agents configuration:

```json
{
  "agents": {
    "crush": {
      "command": "crush",
      "args": ["acp"]
    }
  }
}
```

3. Select Crush as your agent from the agent picker in Zed

### Other ACP-Compatible Editors

Any editor that implements the ACP client protocol can use Crush. Check your editor's documentation for agent configuration instructions.

## Features

When running in ACP mode, Crush provides:

- **Session Management**: Create and manage multiple coding sessions
- **Prompt Processing**: Handle user prompts and generate AI-powered responses
- **Streaming Responses**: Real-time streaming of agent output
- **Cancellation**: Cancel ongoing operations
- **Context Integration**: Full access to Crush's LSP and MCP capabilities

## Protocol Details

Crush implements ACP v1.0.0 with the following capabilities:

### Supported Methods

- `initialize` - Establish connection and negotiate capabilities
- `authenticate` - Optional authentication (currently a no-op)
- `session/new` - Create a new coding session
- `session/load` - Load an existing session
- `session/prompt` - Send a user prompt to the agent

### Supported Notifications

- `session/update` - Stream updates to the client
- `session/cancel` - Cancel ongoing operations

### Content Types

- Text content blocks
- (Future: Images, resources, and other content types)

## Troubleshooting

### Agent Not Responding

1. Check that Crush is properly installed and in your PATH
2. Run `crush acp -d` directly in a terminal to see error messages
3. Check the logs at `./.crush/logs/crush.log`

### Configuration Issues

Ensure your Crush configuration is valid:

```bash
crush schema > crush.json
```

Then validate your configuration against the schema.

### Connection Issues

The ACP protocol requires:
- stdin/stdout to be connected (not redirected)
- JSON-RPC 2.0 compliant messages
- Proper line-delimited JSON format

## Development

### Testing ACP Integration

Run the ACP integration tests:

```bash
go test ./internal/acp -v
```

### Manual Testing

You can manually test the ACP protocol using a simple client:

```bash
# Send an initialize request
echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"1.0.0"}}' | crush acp
```

## Resources

- [Agent Client Protocol Specification](https://agentclientprotocol.com/protocol/overview)
- [ACP GitHub Repository](https://github.com/agentclientprotocol/agent-client-protocol)
- [Crush Documentation](https://github.com/charmbracelet/crush)

## Contributing

If you encounter issues with ACP integration or have suggestions for improvements, please:

1. Check existing issues: https://github.com/charmbracelet/crush/issues
2. Open a new issue with details about your use case
3. Consider contributing a PR with improvements

## License

Crush's ACP integration is part of Crush and is licensed under the FSL-1.1-MIT license.

## Slack GitHub Actions integration

This sends notification to Slack channels, when a workflow of the subscribed
repositories completed.

This is needed because official Slack integration does not support workflow
notification yet. (https://github.com/integrations/slack/issues/940)

Usage:
- `/gha subscribe org/repo`
- `/gha unsubscribe org/repo`

Receive GitHub webhook `workflow_run` events at `:8080`.

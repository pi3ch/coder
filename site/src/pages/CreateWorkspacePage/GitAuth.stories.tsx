import { GitAuth } from "./GitAuth";
import type { Meta, StoryObj } from "@storybook/react";

const meta: Meta<typeof GitAuth> = {
  title: "components/GitAuth",
  component: GitAuth,
};

export default meta;
type Story = StoryObj<typeof GitAuth>;

export const GithubNotAuthenticated: Story = {
  args: {
    type: "github",
    authenticated: false,
  },
};

export const GithubAuthenticated: Story = {
  args: {
    type: "github",
    authenticated: true,
  },
};

export const GitlabNotAuthenticated: Story = {
  args: {
    type: "gitlab",
    authenticated: false,
  },
};

export const GitlabAuthenticated: Story = {
  args: {
    type: "gitlab",
    authenticated: true,
  },
};

export const AzureDevOpsNotAuthenticated: Story = {
  args: {
    type: "azure-devops",
    authenticated: false,
  },
};

export const AzureDevOpsAuthenticated: Story = {
  args: {
    type: "azure-devops",
    authenticated: true,
  },
};

export const BitbucketNotAuthenticated: Story = {
  args: {
    type: "bitbucket",
    authenticated: false,
  },
};

export const BitbucketAuthenticated: Story = {
  args: {
    type: "bitbucket",
    authenticated: true,
  },
};

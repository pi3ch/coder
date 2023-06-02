import Link from "@mui/material/Link"
import { Workspace } from "api/typesGenerated"
import { Maybe } from "components/Conditionals/Maybe"
import { PaginationWidgetBase } from "components/PaginationWidget/PaginationWidgetBase"
import { ComponentProps, FC } from "react"
import { Link as RouterLink } from "react-router-dom"
import { Margins } from "components/Margins/Margins"
import {
  PageHeader,
  PageHeaderSubtitle,
  PageHeaderTitle,
} from "components/PageHeader/PageHeader"
import { Stack } from "components/Stack/Stack"
import { WorkspaceHelpTooltip } from "components/Tooltips"
import { WorkspacesTable } from "components/WorkspacesTable/WorkspacesTable"
import { useLocalStorage } from "hooks"
import difference from "lodash/difference"
import { ImpendingDeletionBanner, Count } from "components/WorkspaceDeletion"
import { ErrorAlert } from "components/Alert/ErrorAlert"
import { Filter } from "./filter/filter"
import { hasError, isApiValidationError } from "api/errors"
import { workspaceFilterQuery } from "utils/filters"
import { SearchBarWithFilter } from "components/SearchBarWithFilter/SearchBarWithFilter"
import Box from "@mui/material/Box"
import Skeleton from "@mui/material/Skeleton"

export const Language = {
  pageTitle: "Workspaces",
  yourWorkspacesButton: "Your workspaces",
  allWorkspacesButton: "All workspaces",
  runningWorkspacesButton: "Running workspaces",
  createANewWorkspace: `Create a new workspace from a `,
  template: "Template",
}

const presetFilters = [
  { query: workspaceFilterQuery.me, name: Language.yourWorkspacesButton },
  { query: workspaceFilterQuery.all, name: Language.allWorkspacesButton },
  {
    query: workspaceFilterQuery.running,
    name: Language.runningWorkspacesButton,
  },
  {
    query: workspaceFilterQuery.failed,
    name: "Failed workspaces",
  },
]

export interface WorkspacesPageViewProps {
  error: unknown
  workspaces?: Workspace[]
  count?: number
  useNewFilter?: boolean
  page: number
  limit: number
  filterProps: ComponentProps<typeof Filter>
  onPageChange: (page: number) => void
  onUpdateWorkspace: (workspace: Workspace) => void
}

export const WorkspacesPageView: FC<
  React.PropsWithChildren<WorkspacesPageViewProps>
> = ({
  workspaces,
  error,
  limit,
  count,
  filterProps,
  onPageChange,
  onUpdateWorkspace,
  useNewFilter,
  page,
}) => {
  const { saveLocal, getLocal } = useLocalStorage()

  const workspaceIdsWithImpendingDeletions = workspaces
    ?.filter((workspace) => workspace.deleting_at)
    .map((workspace) => workspace.id)

  /**
   * Returns a boolean indicating if there are workspaces that have been
   * recently marked for deletion but are not in local storage.
   * If there are, we want to alert the user so they can potentially take action
   * before deletion takes place.
   * @returns {boolean}
   */
  const isNewWorkspacesImpendingDeletion = (): boolean => {
    const dismissedList = getLocal("dismissedWorkspaceList")
    if (!dismissedList) {
      return true
    }

    const diff = difference(
      workspaceIdsWithImpendingDeletions,
      JSON.parse(dismissedList),
    )

    return diff && diff.length > 0
  }

  return (
    <Margins>
      <PageHeader>
        <PageHeaderTitle>
          <Stack direction="row" spacing={1} alignItems="center">
            <span>{Language.pageTitle}</span>
            <WorkspaceHelpTooltip />
          </Stack>
        </PageHeaderTitle>

        <PageHeaderSubtitle>
          {Language.createANewWorkspace}
          <Link component={RouterLink} to="/templates">
            {Language.template}
          </Link>
          .
        </PageHeaderSubtitle>
      </PageHeader>

      <Stack>
        <Maybe condition={hasError(error) && !isApiValidationError(error)}>
          <ErrorAlert error={error} />
        </Maybe>
        {/* <ImpendingDeletionBanner/> determines its own visibility */}
        <ImpendingDeletionBanner
          workspace={workspaces?.find((workspace) => workspace.deleting_at)}
          shouldRedisplayBanner={isNewWorkspacesImpendingDeletion()}
          onDismiss={() =>
            saveLocal(
              "dismissedWorkspaceList",
              JSON.stringify(workspaceIdsWithImpendingDeletions),
            )
          }
          count={Count.Multiple}
        />

        {useNewFilter ? (
          <Filter error={error} {...filterProps} />
        ) : (
          <SearchBarWithFilter
            filter={filterProps.filter.query}
            onFilter={filterProps.filter.debounceUpdate}
            presetFilters={presetFilters}
            error={error}
          />
        )}
      </Stack>

      <Box
        sx={{
          fontSize: 13,
          mb: 2,
          mt: 1,
          color: (theme) => theme.palette.text.secondary,
          "& strong": { color: (theme) => theme.palette.text.primary },
        }}
      >
        {workspaces ? (
          <>
            Showing <strong>{workspaces?.length}</strong> of{" "}
            <strong>{count}</strong> workspaces
          </>
        ) : (
          <Box sx={{ height: 24, display: "flex", alignItems: "center" }}>
            <Skeleton variant="text" width={160} height={16} />
          </Box>
        )}
      </Box>

      <WorkspacesTable
        workspaces={workspaces}
        isUsingFilter={
          filterProps.filter.query !== "" &&
          filterProps.filter.query !== workspaceFilterQuery.me
        }
        onUpdateWorkspace={onUpdateWorkspace}
        error={error}
      />
      {count !== undefined && (
        <PaginationWidgetBase
          count={count}
          limit={limit}
          onChange={onPageChange}
          page={page}
        />
      )}
    </Margins>
  )
}

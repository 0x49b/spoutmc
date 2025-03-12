import { createRoot } from "react-dom/client";
import "@patternfly/react-core/dist/styles/base.css";
import "./fonts.css";
import { Icon } from "@patternfly/react-core";
import LongArrowAltDownIcon from "@patternfly/react-icons/dist/esm/icons/long-arrow-alt-down-icon";

/* eslint-disable no-console */
import React from "react";
import {
  Button,
  MenuToggle,
  ToggleGroup,
  ToggleGroupItem,
  ToggleGroupItemProps,
} from "@patternfly/react-core";
import {
  Table,
  TableText,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
  CustomActionsToggleProps,
  ActionsColumn,
  IAction,
} from "@patternfly/react-table";

interface Repository {
  name: string;
  branches: string;
  prs: string;
  workspaces: string;
  lastCommit: string;
  singleAction: string;
}

type ExampleType = "defaultToggle" | "customToggle";

export const TableActions: React.FunctionComponent = () => {
  // In real usage, this data would come from some external source like an API via props.
  const repositories: Repository[] = [
    {
      name: "one",
      branches: "two",
      prs: "a",
      workspaces: "four",
      lastCommit: "five",
      singleAction: "Start",
    },
  ];

  const columnNames = {
    name: "Repositories",
    branches: "Branches",
    prs: "Pull requests",
    workspaces: "Workspaces",
    lastCommit: "Last commit",
    singleAction: "Single action",
  };

  // This state is just for the ToggleGroup in this example and isn't necessary for Table usage.
  const [exampleChoice, setExampleChoice] =
    React.useState<ExampleType>("defaultToggle");
  const onExampleTypeChange: ToggleGroupItemProps["onChange"] = (
    event,
    _isSelected
  ) => {
    const id = event.currentTarget.id;
    setExampleChoice(id as ExampleType);
  };

  const customActionsToggle = (props: CustomActionsToggleProps) => (
    <MenuToggle
      ref={props.toggleRef}
      onClick={props.onToggle}
      isDisabled={props.isDisabled}
    >
      Actions
    </MenuToggle>
  );

  const defaultActions = (repo: Repository): IAction[] => [
    {
      title: "Delete",
      onClick: () => console.log(`clicked on Some action, on row ${repo.name}`),
    },
  ];

  return (
    <React.Fragment>
      <Table aria-label="Actions table">
        <Thead>
          <Tr>
            <Th>{columnNames.name}</Th>
            <Th>{columnNames.branches}</Th>
            <Th>{columnNames.prs}</Th>
            <Th>{columnNames.workspaces}</Th>
            <Th>{columnNames.lastCommit}</Th>
            <Th screenReaderText="Primary action" />
            <Th screenReaderText="Secondary action" />
          </Tr>
        </Thead>
        <Tbody>
          {repositories.map((repo) => {
            // Arbitrary logic to determine which rows get which actions in this example
            let rowActions: IAction[] | null = defaultActions(repo);

            return (
              <Tr key={repo.name}>
                <Td dataLabel={columnNames.name}>{repo.name}</Td>
                <Td dataLabel={columnNames.branches}>{repo.branches}</Td>
                <Td dataLabel={columnNames.prs}>{repo.prs}</Td>
                <Td dataLabel={columnNames.workspaces}>{repo.workspaces}</Td>
                <Td dataLabel={columnNames.lastCommit}>{repo.lastCommit}</Td>
                <Td dataLabel={columnNames.singleAction} modifier="fitContent">
                  <TableText>
                    <Button variant="secondary" size="sm">
                      <Icon>
                        <LongArrowAltDownIcon />
                      </Icon>
                    </Button>
                    <Button variant="primary" size="sm">
                      Stop
                    </Button>
                  </TableText>
                </Td>
                <Td isActionCell>
                  {" "}
                  {rowActions ? <ActionsColumn items={rowActions} /> : null}
                </Td>
              </Tr>
            );
          })}
        </Tbody>
      </Table>
    </React.Fragment>
  );
};

const container = document.getElementById("root");
createRoot(container).render(<TableActions />);

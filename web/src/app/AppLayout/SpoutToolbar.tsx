import * as React from "react";
import {
  Avatar,
  Button,
  ButtonVariant,
  Dropdown,
  DropdownItem,
  DropdownList,
  MenuToggle,
  MenuToggleElement,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem
} from "@patternfly/react-core";
import {QuestionCircleIcon} from "@patternfly/react-icons";
import spoutmclogo from "@app/bgimages/Logo.svg";
import ConnectionState from "@app/AppLayout/ConnectionState";


const SpoutToolbar: React.FunctionComponent = () => {
  const [isDropdownOpen, setIsDropdownOpen] = React.useState(false);

  const onDropdownToggle = () => {
    setIsDropdownOpen(!isDropdownOpen);
  };

  const onDropdownSelect = () => {
    setIsDropdownOpen(!isDropdownOpen);
  };

  return (
    <Toolbar id="toolbar" isStatic>
      <ToolbarContent>
        <ToolbarGroup
          variant="action-group-plain"
          align={{default: 'alignEnd'}}
          gap={{default: 'gapNone', md: 'gapMd'}}
        >
          {/*<ToolbarItem>
            <Button aria-label="Notifications" variant={ButtonVariant.plain} icon={<BellIcon/>}/>
          </ToolbarItem>*/}

          <ToolbarGroup variant="action-group-plain" visibility={{default: 'hidden', lg: 'visible'}}>
            <ToolbarItem>
              <ConnectionState/>
            </ToolbarItem>
          </ToolbarGroup>
        </ToolbarGroup>
        <ToolbarItem visibility={{default: 'hidden', md: 'visible'}}>
          <Dropdown
            isOpen={isDropdownOpen}
            onSelect={onDropdownSelect}
            onOpenChange={(isOpen: boolean) => setIsDropdownOpen(isOpen)}
            popperProps={{position: 'right'}}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
              <MenuToggle
                ref={toggleRef}
                onClick={onDropdownToggle}
                isExpanded={isDropdownOpen}
                icon={<Avatar src={spoutmclogo} alt="" size="sm"/>}
              >
                Ned Username
              </MenuToggle>
            )}
          >
            <DropdownList>
              <DropdownItem key="group 2 profile">My profile</DropdownItem>
              <DropdownItem key="group 2 user">User management</DropdownItem>
              <DropdownItem key="group 2 logout">Logout</DropdownItem>
            </DropdownList>
          </Dropdown>
        </ToolbarItem>
      </ToolbarContent>
    </Toolbar>
  );
}

export default SpoutToolbar;

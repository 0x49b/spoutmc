import React, { useMemo, useState } from 'react';
import {
  DualListSelector,
  DualListSelectorPane,
  DualListSelectorList,
  DualListSelectorListItem,
  DualListSelectorControlsWrapper,
  DualListSelectorControl,
  SearchInput
} from '@patternfly/react-core';
import {
  AngleDoubleLeftIcon,
  AngleDoubleRightIcon,
  AngleLeftIcon,
  AngleRightIcon
} from '@patternfly/react-icons';

export interface PermissionOption {
  id: number;
  key: string;
  description?: string;
}

export interface RolePermissionsDualListProps {
  id: string;
  allPermissions: PermissionOption[];
  chosenIds: number[];
  onChosenIdsChange: (ids: number[]) => void;
  isDisabled?: boolean;
  availableTitle?: string;
  chosenTitle?: string;
}

/**
 * PatternFly Dual list selector: left = available, right = chosen (by permission id).
 * @see https://www.patternfly.org/components/dual-list-selector
 */
export const RolePermissionsDualList: React.FC<RolePermissionsDualListProps> = ({
  id,
  allPermissions,
  chosenIds,
  onChosenIdsChange,
  isDisabled = false,
  availableTitle = 'Available permissions',
  chosenTitle = 'Assigned permissions'
}) => {
  const chosenSet = useMemo(() => new Set(chosenIds), [chosenIds]);
  const [leftSearch, setLeftSearch] = useState('');
  const [rightSearch, setRightSearch] = useState('');
  const [selectedLeft, setSelectedLeft] = useState<string[]>([]);
  const [selectedRight, setSelectedRight] = useState<string[]>([]);

  const available = useMemo(() => {
    const q = leftSearch.trim().toLowerCase();
    return allPermissions
      .filter((p) => !chosenSet.has(p.id))
      .filter(
        (p) =>
          !q ||
          p.key.toLowerCase().includes(q) ||
          (p.description || '').toLowerCase().includes(q)
      );
  }, [allPermissions, chosenSet, leftSearch]);

  const chosen = useMemo(() => {
    const q = rightSearch.trim().toLowerCase();
    return allPermissions
      .filter((p) => chosenSet.has(p.id))
      .filter(
        (p) =>
          !q ||
          p.key.toLowerCase().includes(q) ||
          (p.description || '').toLowerCase().includes(q)
      );
  }, [allPermissions, chosenSet, rightSearch]);

  const toggleLeft = (_e: React.MouseEvent | React.ChangeEvent | React.KeyboardEvent, itemId?: string) => {
    if (!itemId) return;
    setSelectedLeft((prev) =>
      prev.includes(itemId) ? prev.filter((x) => x !== itemId) : [...prev, itemId]
    );
  };

  const toggleRight = (_e: React.MouseEvent | React.ChangeEvent | React.KeyboardEvent, itemId?: string) => {
    if (!itemId) return;
    setSelectedRight((prev) =>
      prev.includes(itemId) ? prev.filter((x) => x !== itemId) : [...prev, itemId]
    );
  };

  const moveToChosen = () => {
    const add = selectedLeft.map((s) => parseInt(s, 10)).filter((n) => !Number.isNaN(n));
    const next = new Set(chosenIds);
    add.forEach((n) => next.add(n));
    onChosenIdsChange(Array.from(next).sort((a, b) => a - b));
    setSelectedLeft([]);
  };

  const moveToAvailable = () => {
    const remove = new Set(selectedRight.map((s) => parseInt(s, 10)));
    onChosenIdsChange(chosenIds.filter((id) => !remove.has(id)));
    setSelectedRight([]);
  };

  const moveAllToChosen = () => {
    const availIds = allPermissions.filter((p) => !chosenSet.has(p.id)).map((p) => p.id);
    const next = new Set([...chosenIds, ...availIds]);
    onChosenIdsChange(Array.from(next).sort((a, b) => a - b));
    setSelectedLeft([]);
  };

  const moveAllToAvailable = () => {
    onChosenIdsChange([]);
    setSelectedRight([]);
  };

  const leftStatus = `${selectedLeft.length} of ${available.length} options selected`;
  const rightStatus = `${selectedRight.length} of ${chosen.length} options selected`;

  return (
    <DualListSelector id={id}>
      <DualListSelectorPane
        title={availableTitle}
        status={leftStatus}
        searchInput={
          <SearchInput
            placeholder="Search available"
            value={leftSearch}
            onChange={(_e, v) => setLeftSearch(v)}
            onClear={() => setLeftSearch('')}
            isDisabled={isDisabled}
          />
        }
        isDisabled={isDisabled}
      >
        <DualListSelectorList>
          {available.map((p) => (
            <DualListSelectorListItem
              key={p.id}
              id={String(p.id)}
              isSelected={selectedLeft.includes(String(p.id))}
              onOptionSelect={toggleLeft}
              isDisabled={isDisabled}
            >
              <span title={p.description}>{p.key}</span>
            </DualListSelectorListItem>
          ))}
        </DualListSelectorList>
      </DualListSelectorPane>

      <DualListSelectorControlsWrapper aria-label="Permission move controls">
        <DualListSelectorControl
          icon={<AngleRightIcon />}
          aria-label="Move selected to assigned"
          tooltipContent="Move selected to assigned"
          onClick={moveToChosen}
          isDisabled={isDisabled || selectedLeft.length === 0}
        />
        <DualListSelectorControl
          icon={<AngleDoubleRightIcon />}
          aria-label="Move all to assigned"
          tooltipContent="Move all to assigned"
          onClick={moveAllToChosen}
          isDisabled={isDisabled || available.length === 0}
        />
        <DualListSelectorControl
          icon={<AngleLeftIcon />}
          aria-label="Move selected to available"
          tooltipContent="Move selected to available"
          onClick={moveToAvailable}
          isDisabled={isDisabled || selectedRight.length === 0}
        />
        <DualListSelectorControl
          icon={<AngleDoubleLeftIcon />}
          aria-label="Remove all from assigned"
          tooltipContent="Remove all from assigned"
          onClick={moveAllToAvailable}
          isDisabled={isDisabled || chosenIds.length === 0}
        />
      </DualListSelectorControlsWrapper>

      <DualListSelectorPane
        isChosen
        title={chosenTitle}
        status={rightStatus}
        searchInput={
          <SearchInput
            placeholder="Search assigned"
            value={rightSearch}
            onChange={(_e, v) => setRightSearch(v)}
            onClear={() => setRightSearch('')}
            isDisabled={isDisabled}
          />
        }
        isDisabled={isDisabled}
      >
        <DualListSelectorList>
          {chosen.map((p) => (
            <DualListSelectorListItem
              key={p.id}
              id={String(p.id)}
              isSelected={selectedRight.includes(String(p.id))}
              onOptionSelect={toggleRight}
              isDisabled={isDisabled}
            >
              <span title={p.description}>{p.key}</span>
            </DualListSelectorListItem>
          ))}
        </DualListSelectorList>
      </DualListSelectorPane>
    </DualListSelector>
  );
};

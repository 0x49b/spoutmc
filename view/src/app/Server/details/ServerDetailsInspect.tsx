import React from 'react';
import {
  Button,
  ClipboardCopyButton,
  CodeBlock,
  CodeBlockAction,
  CodeBlockCode,
  ExpandableSection,
  ExpandableSectionToggle
} from '@patternfly/react-core';
import {RedoIcon} from "@patternfly/react-icons";

type ServerDetailsInspectProps = {
  inspectJson: string
}

export const ServerDetailsInspect: React.FC<ServerDetailsInspectProps> = ({inspectJson}) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const onToggle = (isExpanded) => {
    setIsExpanded(isExpanded);
  };

  const clipboardCopyFunc = (event, text) => {
    navigator.clipboard.writeText(text.toString());
  };

  const onClickCopyButton = (event, text) => {
    clipboardCopyFunc(event, text);
    setCopied(true);
  };

  const reloadInspectData = () =>{
    // todo implement this
  }

  const splitAfterNLines = (input: string, n: number): [string, string] => {
    if (!input) return ['', ''];
    const lines = input.split('\n');
    const firstChunk = lines.slice(0, n).join('\n');
    const secondChunk = lines.slice(n).join('\n');
    return [firstChunk, secondChunk];
  }

  const splittedJsonInspect = splitAfterNLines(inspectJson, 15)
  const copyBlock = inspectJson
  const code = splittedJsonInspect[0]
  const expandedCode = splittedJsonInspect[1]


  const actions = (
    <>
      <CodeBlockAction>
        <ClipboardCopyButton
          id="expandable-copy-button"
          textId="code-content"
          aria-label="Copy to clipboard"
          onClick={(e) => onClickCopyButton(e, copyBlock)}
          exitDelay={copied ? 1500 : 600}
          maxWidth="110px"
          variant="plain"
          onTooltipHidden={() => setCopied(false)}
        >
          {copied ? 'Successfully copied to clipboard!' : 'Copy to clipboard'}
        </ClipboardCopyButton>
      </CodeBlockAction>
      <CodeBlockAction>
        <Button variant="plain" aria-label="Reload Inspect Data" icon={<RedoIcon/>}/>
      </CodeBlockAction>
    </>
  );

  return (
    <>
      <CodeBlock actions={actions}>
        <CodeBlockCode>
          {code}
          <ExpandableSection isExpanded={isExpanded}
                             isDetached
                             contentId="code-block-expand"
                             toggleId="code-block-toggle">
            {expandedCode}
          </ExpandableSection>
        </CodeBlockCode>
        <ExpandableSectionToggle isExpanded={isExpanded}
                                 onToggle={onToggle}
                                 contentId="code-block-expand"
                                 direction="up"
                                 toggleId="code-block-toggle">
          {isExpanded ? 'Show Less' : 'Show More'}
        </ExpandableSectionToggle>
      </CodeBlock>
    </>
  );
};

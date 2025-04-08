import React, { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { useServerWebSocket } from '../../../services/websocketService';
import { CodeEditor } from '@patternfly/react-code-editor';
import { WsCommand, WsCommandType } from '@app/model/wsCommand';
import { useSelector } from 'react-redux';
import { RootState } from '@app/store/store';
import {
  Button,
  Content,
  Flex,
  FlexItem,
  Grid,
  GridItem,
  HelperText,
  HelperTextItem,
  Icon
} from '@patternfly/react-core';
import { PaperPlaneIcon } from '@patternfly/react-icons';
import CommandAutocomplete from '@app/Server/details/components/commandAutocomplete/CommandAutocomplete';

export const ServerDetailsLogs: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  const { sendMessage } = useServerWebSocket();
  const serverLogs = useSelector((state: RootState) => state.server.serverLogs);
  const [logContent, setLogContent] = useState<string>('');
  const [command, setCommand] = useState<string>('');

  const editorRef = useRef<any>(null); // Ref to Monaco editor instance

  useEffect(() => {
    loadServerLogs();
  }, [serverId]);

  useEffect(() => {
    if (serverId) {
      if (Array.isArray(serverLogs[serverId])) {
        // @ts-ignore
        const newLog = serverLogs[serverId].join('\n');
        setLogContent(newLog);
        scrollToBottom();
      }
    }
  }, [serverLogs]);

  const scrollToBottom = () => {
    setTimeout(() => {
      if (editorRef.current) {
        const model = editorRef.current.getModel();
        const lineCount = model.getLineCount();
        editorRef.current.revealLine(lineCount);
      }
    }, 0);
  };

  const loadServerLogs = () => {
    const commandMessage: WsCommand = {
      type: WsCommandType.LOGS,
      containerId: serverId
    };
    sendMessage(JSON.stringify(commandMessage));
  };

  const onEditorDidMount = (editor, monaco) => {
    editor.layout();
    monaco.editor.getModels()[0].updateOptions({ tabSize: 5 });
    editorRef.current = editor; // Store the editor instance
    if (logContent) {
      scrollToBottom();
    }
  };

  const onChange = (value) => {

  };

  return (

    <Flex direction={{ default: 'column' }}>
      <FlexItem>
        <CodeEditor
          isDarkTheme={true}
          isLineNumbersVisible={true}
          isReadOnly={true}
          isMinimapVisible={false}
          code={logContent}
          onChange={onChange}
          onEditorDidMount={onEditorDidMount}
          height="400px"
        />
      </FlexItem>
      <FlexItem>

        <Grid hasGutter>
          <GridItem span={1}><Content><b>Server Command</b></Content></GridItem>
          <GridItem span={10}>
            <Flex direction={{ default: 'column' }}>
              <FlexItem>


                {/*<TextInput
                  value={command}
                  type="text"
                  onChange={(_event, value) => setCommand(value)}
                  aria-label="Server command input box"
                />*/}
                <CommandAutocomplete onComplete={(value) => console.log('Final command:', value)} />



              </FlexItem>
              <FlexItem>
                <HelperText>
                  <HelperTextItem>
                    Send Command by pressing <kbd>Enter</kbd> or the <Icon><PaperPlaneIcon /></Icon>
                  </HelperTextItem>
                </HelperText>
              </FlexItem>
            </Flex>
          </GridItem>
          <GridItem span={1}>
            <Button variant="control" icon={<PaperPlaneIcon />} />
          </GridItem>
        </Grid>

      </FlexItem>
    </Flex>
  );
};

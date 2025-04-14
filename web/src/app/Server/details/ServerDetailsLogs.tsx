import React, { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
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
  Icon,
  TextInput
} from '@patternfly/react-core';
import { PaperPlaneIcon } from '@patternfly/react-icons';
import { ReadyState } from 'react-use-websocket';
import { useSharedWebSocket } from '@app/connection/WebSocketContext';


export const ServerDetailsLogs: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  const { sendMessage, readyState } = useSharedWebSocket();
  const serverLogs = useSelector((state: RootState) => state.server.serverLogs);
  const [logContent, setLogContent] = useState<string>('');
  const [serverCommand, setServerCommand] = useState<string>('');

  const editorRef = useRef<any>(null); // Ref to Monaco editor instance


  useEffect(() => {
    if (readyState === ReadyState.OPEN) {
      loadServerLogs();
    }
  }, [readyState, serverId]);

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
    if (readyState !== ReadyState.OPEN) {
      console.error('WebSocket not open');
      return;
    }
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

  const onChangeCommand = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
    setServerCommand(value);
  };

  const sendCommand = () => {
    if (!serverCommand) return;

    if (readyState !== ReadyState.OPEN) {
      console.warn('WebSocket is not open. Cannot send command yet.');
      // Optionally show an error or UI message here
      return;
    }

    const commandMessage: WsCommand = {
      type: WsCommandType.EXEC_REQUEST,
      containerId: serverId,
      message: serverCommand
    };

    console.log(JSON.stringify(commandMessage));

    sendMessage(JSON.stringify(commandMessage));
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      sendCommand();
    }
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

                {/*<CommandAutocomplete onComplete={(value) => handleCommand(value)} />*/}
                <TextInput
                  id={serverId}
                  value={serverCommand}
                  onChange={onChangeCommand}
                  onKeyDown={handleKeyDown}
                  isDisabled={readyState !== ReadyState.OPEN}
                />

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
            <Button variant="control" icon={<PaperPlaneIcon />} onClick={() => sendCommand()} />
          </GridItem>
        </Grid>

      </FlexItem>
    </Flex>
  );
};

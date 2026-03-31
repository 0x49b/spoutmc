import React from 'react';
import {
    Alert,
    Button,
    ButtonVariant,
    Grid,
    GridItem,
    Modal,
    ModalBody,
    ModalFooter,
    ModalVariant
} from '@patternfly/react-core';

interface RestartServerModalPropsProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    serverName: string;
    isLoading?: boolean;
}

const RestartServerModalProps: React.FC<RestartServerModalPropsProps> = ({
                                                                             isOpen,
                                                                             onClose,
                                                                             onConfirm,
                                                                             serverName,
                                                                             isLoading = false
                                                                         }) => {
    return (
        <Modal
            variant={ModalVariant.small}
            isOpen={isOpen}
            onClose={onClose}
            onEscapePress={onClose}>
            <ModalBody>
                <Grid hasGutter component="ul">
                    <GridItem className="pf-v6-u-mt-xl">Are you sure you want to restart
                        Server <strong>{serverName}</strong>?</GridItem>

                    <Alert variant="danger"
                           title="All players will be disconnected when the server restarts."
                           ouiaId="DisconnectPlayerAlert"/>

                </Grid>
            </ModalBody>
            <ModalFooter>
                <Button variant={ButtonVariant.danger}
                        isLoading={isLoading}
                        isDisabled={isLoading}
                        onClick={onConfirm}>
                    {isLoading ? 'Restarting...' : 'Restart Server'}
                </Button>
                <Button variant={ButtonVariant.link} onClick={onClose} isDisabled={isLoading}>
                    cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default RestartServerModalProps;

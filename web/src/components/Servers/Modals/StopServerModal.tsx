import React from 'react';
import {
    Button,
    ButtonVariant,
    Grid,
    GridItem,
    Icon,
    Modal,
    ModalBody,
    ModalFooter,
    ModalVariant
} from '@patternfly/react-core';
import {ExclamationTriangleIcon} from "@patternfly/react-icons";

interface StopServerModalProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    serverName: string;
    isLoading?: boolean;
}

const StopServerModal: React.FC<StopServerModalProps> = ({
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
                    <GridItem className="pf-v6-u-mt-xl">Are you sure you want to stop
                        Server <strong>{serverName}</strong>?</GridItem>
                    <GridItem>
                        <Icon status="warning" className="mr-2">
                            <ExclamationTriangleIcon/>
                        </Icon>
                        All players will be disconnected when the server stops.
                    </GridItem>
                </Grid>
            </ModalBody>
            <ModalFooter>
                <Button variant={ButtonVariant.danger}
                        isLoading={isLoading}
                        isDisabled={isLoading}
                        onClick={onConfirm}>
                    {isLoading ? 'Stopping...' : 'Stop Server'}
                </Button>
                <Button variant={ButtonVariant.link} onClick={onClose} isDisabled={isLoading}>
                    cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default StopServerModal;

import React, {useState} from 'react';
import {
    Button,
    Checkbox,
    Grid,
    GridItem,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant
} from '@patternfly/react-core';
import {TrashIcon} from '@patternfly/react-icons';

interface DeleteServerModalProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (removeData: boolean) => void;
    serverName: string;
    isLoading?: boolean;
}

const DeleteServerModal: React.FC<DeleteServerModalProps> = ({
                                                                 isOpen,
                                                                 onClose,
                                                                 onConfirm,
                                                                 serverName,
                                                                 isLoading = false
                                                             }) => {
    const [removeData, setRemoveData] = useState(true);

    const handleConfirm = () => {
        onConfirm(removeData);
    };

    return (
        <Modal
            variant={ModalVariant.small}
            isOpen={isOpen}
            onClose={onClose}>
            <ModalHeader title="Delete Server" titleIconVariant={"warning"}/>
            <ModalBody>
                <Grid hasGutter component="ul">
                    <GridItem>Are you sure you want to
                        delete <strong>{serverName}</strong>?</GridItem>
                    <GridItem>
                        <Checkbox
                            id="removeData"
                            label="Remove all server data"
                            description="Delete all files in the server's data directory (recommended)"
                            isChecked={removeData}
                            onChange={(_event, checked) => setRemoveData(checked)}
                            isDisabled={isLoading}
                        />
                    </GridItem>
                </Grid>
            </ModalBody>
            <ModalFooter>
                <Button
                    key="delete"
                    variant="danger"
                    onClick={handleConfirm}
                    isLoading={isLoading}
                    isDisabled={isLoading}
                    icon={<TrashIcon/>}
                    size="sm"
                >
                    {isLoading ? 'Deleting...' : 'Delete Server'}
                </Button>
                <Button key="cancel" variant="link" onClick={onClose} isDisabled={isLoading}
                        size="sm">
                    Cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default DeleteServerModal;

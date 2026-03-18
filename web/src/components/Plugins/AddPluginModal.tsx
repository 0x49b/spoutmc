import React from 'react';
import {
    Button,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant
} from '@patternfly/react-core';

interface AddPluginModalProps {
    isOpen: boolean;
    onClose: () => void;
}

const AddPluginModal: React.FC<AddPluginModalProps> = ({isOpen, onClose}) => {
    return (
        <Modal
            variant={ModalVariant.small}
            title="Add Plugin"
            isOpen={isOpen}
            onClose={onClose}
        >
            <ModalHeader>Add Plugin</ModalHeader>
            <ModalBody><p>Plugin installation form goes here</p></ModalBody>
            <ModalFooter>
                <Button key="add" variant="primary" onClick={onClose}>Add</Button>,
                <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
            </ModalFooter>

        </Modal>
    );
};

export default AddPluginModal;

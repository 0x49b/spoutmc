import React from 'react';
import {Flex, FlexItem, PageSection, Title} from '@patternfly/react-core';
import Breadcrumb from './Breadcrumb';

interface PageHeaderProps {
    title: string,
    description?: string,
    actions?: React.ReactNode,
    serverStatus: React.ReactNode
}

export const PageHeader: React.FC<PageHeaderProps> = ({
    title,
    description,
    actions,
    serverStatus,
}) => {
    return (
        <>
            <PageSection variant="default" className="pf-v6-u-pb-0">
                <Breadcrumb />
            </PageSection>
            <PageSection variant="default">
                <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }} alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem>
                        <Title headingLevel="h1" size="2xl">{title} {serverStatus}</Title>

                        {description && (
                            <p className="pf-v6-u-color-200 pf-v6-u-mt-sm">{description}</p>
                        )}
                    </FlexItem>
                    {actions && (
                        <FlexItem>
                            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                {actions}
                            </Flex>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
        </>
    );
};

export default PageHeader;

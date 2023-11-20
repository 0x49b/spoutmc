import './App.css';
import {useDisclosure} from '@mantine/hooks';
import {AppShell, Burger, Divider, Group, NavLink, Title, useMantineTheme} from '@mantine/core';
import React, {CSSProperties} from "react";
import spoutSVG from '@/assets/spout.svg'
import {Outlet} from 'react-router-dom';
import {IconServer} from '@tabler/icons-react';


const styles: { [key: string]: CSSProperties } = {
    container: {
        display: "flex",
        alignItems: "center",
        height: "100%",
        justifyContent: "space-between",
    },
};

function App() {
    const [opened, {toggle}] = useDisclosure();
    const theme = useMantineTheme();

    return (
        <AppShell
            header={{height: 60}}
            navbar={{width: 300, breakpoint: 'sm', collapsed: {mobile: !opened}}}
            padding="md"
        >
            <AppShell.Header>
                <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" color={theme.colors.gray[6]}/>
                <Group>
                    <img src={spoutSVG} alt="SpoutMC Logo" width="36" height="48"/>
                    <Title style={styles.title}>SpoutMC</Title>
                </Group>
            </AppShell.Header>

            <AppShell.Navbar p="md">
                <Title order={3}>Server</Title>
                <NavLink label="Serverlist" leftSection={<IconServer size="1rem" stroke={1.5}/>}/>
                <Divider/>
            </AppShell.Navbar>

            <AppShell.Main>
                Main
                <Outlet/>
            </AppShell.Main>

        </AppShell>
    );
}

export default App;

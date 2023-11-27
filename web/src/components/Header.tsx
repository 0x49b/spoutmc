import {AppShell, Burger, Button, Flex, useComputedColorScheme, useMantineColorScheme} from '@mantine/core';
import {FiMoon, FiSun} from 'react-icons/fi';
import spoutLogo from '../assets/spout.svg'

const Header = ({toggle, opened}: any) => {
    const {setColorScheme} = useMantineColorScheme();
    const computedColorScheme = useComputedColorScheme('light');

    const toggleColorScheme = () => {
        setColorScheme(computedColorScheme === 'dark' ? 'light' : 'dark');
    };

    return (
        <AppShell.Header>
            <Flex justify="space-between" align="center" style={{padding: '10px 20px'}}>
                <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm"/>
                <div><img src={spoutLogo} alt={"SpoutMC Logo"} height={40} width={40}/>SpoutMC</div>
                <Button color="gray" size="sm" radius="xl" variant="outline"
                        onClick={toggleColorScheme}> {computedColorScheme === 'dark' ?
                    <FiSun/> : <FiMoon/>} </Button>
            </Flex>
        </AppShell.Header>
    );
};

export default Header;
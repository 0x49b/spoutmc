import {AppShell, Divider, NavLink, Title} from '@mantine/core';
import {useNavigate} from 'react-router-dom';


const Navbar = () => {
    const navigate = useNavigate();

    return (
        <AppShell.Navbar p="md" style={{gap: '10px'}}>
            <Title ta="left" order={3}>Server</Title>
            <NavLink
                label="Server List"
                onClick={() => navigate('/server')}
                style={{margin: '5px'}}
            />
            <NavLink
                label="Create"
                onClick={() => navigate('/server-create')}
                style={{margin: '5px'}}
            />
            <Divider/>
            <NavLink
                label="Button Component"
                onClick={() => navigate('/button-component')}
                style={{margin: '5px'}}
            />
        </AppShell.Navbar>
    );
};

export default Navbar;
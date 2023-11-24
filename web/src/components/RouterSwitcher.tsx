import {Route, Routes} from 'react-router-dom';
import ButtonComponent from './Buttons';
import ServerList from "@/components/server/List";
import ServerCreate from "@/components/server/Create";

const RouteSwitcher = () => {
    return (
        <Routes>
            <Route path="*" element={<ServerList/>}/>
            <Route path="/server" element={<ServerList/>}/>
            <Route path="/server-create" element={<ServerCreate/>}/>
            <Route path="/button-component" element={<ButtonComponent/>}/>
        </Routes>
    );
};

export default RouteSwitcher;
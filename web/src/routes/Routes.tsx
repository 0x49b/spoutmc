import Servers from "@/servers/Servers";
import {Outlet, Route, Routes} from "react-router-dom";


const AppRoutes = () => {
    return (
        <Routes>
            <Route path="/" element={<Outlet/>}>
                <Route
                    path="Servers"
                    element={<Servers/>}
                />
            </Route>
        </Routes>
    )
}
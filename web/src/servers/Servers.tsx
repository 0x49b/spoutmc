import {useDisclosure} from "@mantine/hooks";
import {useMantineTheme} from "@mantine/core";
import React from "react";

function Servers() {
    const [opened, {toggle}] = useDisclosure();
    const theme = useMantineTheme();

    return (
        <div>SERVERS is working</div>
    );
}

export default Servers;

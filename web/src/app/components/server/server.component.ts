import {Component, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {RouterLink} from "@angular/router";
import {
  ClrDatagridModule,
  ClrDropdownModule,
  ClrInputModule,
  ClrModal,
  ClrModalModule,
  ClrRadioModule,
  ClrSelectModule,
  ClrWizard,
  ClrWizardModule
} from "@clr/angular";
import {MCServerDetail} from "../../model/serverDetail";
import {LoaderComponent} from "../util/loader/loader.component";
import {FormControl, FormGroup, FormsModule, ReactiveFormsModule, Validators} from "@angular/forms";
import {CdsIconModule} from "@cds/angular";
import {OutlineIconsModule} from "@dimaslz/ng-heroicons";
import {RestService} from "../../services/rest/rest.service";
import {ContainerCommand} from "../../model/containerCommand";
import {ServerType} from "../../model/serverType";
import {WebsocketService} from "../../services/websocket/websocket.service";

export interface ReloadTimes {
  value: number,
  viewValue: string
}

@Component({
  selector: 'app-server',
  standalone: true,
  imports: [
    RouterLink,
    ClrDatagridModule,
    LoaderComponent,
    FormsModule,
    ClrSelectModule,
    CdsIconModule,
    OutlineIconsModule,
    ClrModalModule,
    ClrWizardModule,
    ReactiveFormsModule,
    ClrInputModule,
    ClrRadioModule,
    ClrDropdownModule
  ],
  templateUrl: './server.component.html',
  styleUrl: './server.component.css'
})
export class ServerComponent implements OnInit, OnDestroy {

  loading = false
  actionLoading = false
  dataSource: MCServerDetail[] = []
  reloadInterval: any = null
  reload: number = 5
  selected: MCServerDetail[] = []
  RELOAD_STORAGE_KEY = "reloadTime"
  reloadTimes: ReloadTimes[] = [
    {value: 5, viewValue: '5 Seconds'},
    {value: 10, viewValue: '10 Seconds'},
    {value: 20, viewValue: '20 Seconds'},
    {value: 30, viewValue: '30 Seconds'},
    {value: 45, viewValue: '45 Seconds'},
    {value: 60, viewValue: '1 Minute'},
    {value: -1, viewValue: 'never'},
  ];

  // Todo on load check if lobby and proxy exists, in the labels
  proxyServerExists: boolean = false
  lobbyServerExists: boolean = false

  @ViewChild("dialogConfirmation") confirmationDialog?: ClrModal;
  dialogOpen = false
  dialogCommand: ContainerCommand = {
    title: "", body: "", callback: () => {
    }
  }

  @ViewChild("wizardNewServer") wizardMedium?: ClrWizard;
  mdOpen = false
  newServerGeneralForm = new FormGroup({
    servername: new FormControl('', [Validators.required, Validators.minLength(4)]),
    servertype: new FormControl(ServerType.GAME),
  })

  constructor(private restService: RestService, private wsService: WebsocketService) {
    this.reload = parseInt(this.readReloadTime())
  }

  ngOnInit() {

    this.wsService.connect();
    this.wsService.receiveMessage().subscribe((message) => {
      console.log(message)
    })
    this.sendMessage()


    this.loading = true
    this.loadServerData()
    this.initializeInterval()
  }

  sendMessage() {
    this.wsService.sendMessage("bola")
  }

  ngOnDestroy() {
    clearInterval(this.reloadInterval)
  }

  initializeInterval() {
    window.localStorage.setItem(this.RELOAD_STORAGE_KEY, this.reload.toString())
    // Switch Back on when correct reload in place with changing each item
    /*if (this.reload > 0) {
      this.reloadInterval = setInterval(() => {
        this.loadServerData()
      }, this.reload * 1000)
    }*/
  }

  setNewInterval() {
    clearInterval(this.reloadInterval)
    this.initializeInterval()
  }

  readReloadTime() {
    return window.localStorage.getItem(this.RELOAD_STORAGE_KEY) || "5";
  }


  loadServerData() {
    this.restService.getAllServersWithDetails().subscribe({
      next: data => {
        this.dataSource = []
        this.dataSource = data
        this.checkProxyServer(data)
        this.checkLobbyServer(data)
        this.loading = false
      }, error: err => {
      }
    })
  }

  bobSaget() {
  }

  checkForLabel(servers: MCServerDetail[], label: string): boolean {
    servers.forEach(s => {
      return !!s.Config.Labels[label] || false
    })
    return false
  }

  checkProxyServer(servers: MCServerDetail[]) {
    this.proxyServerExists = this.checkForLabel(servers, "io.spout.proxy")
  }

  checkLobbyServer(servers: MCServerDetail[]) {
    this.lobbyServerExists = this.checkForLabel(servers, "io.spout.lobby")
  }


  stopContainer(containerId: string) {
    this.actionLoading = true
    let stopCommand: ContainerCommand = {
      title: "Confirmation",
      body: "Do you want to stop this Server?",
      callback: () => this.restService.stopContainer(containerId).subscribe(
        {
          next: data => {
            this.actionLoading = false
            this.loadServerData()
          },
          error: err => {
          }
        })
    }

    this.showDialog(stopCommand)

  }

  startContainer(containerId: string) {
    this.actionLoading = true
    let startCommand: ContainerCommand = {
      title: "Confirmation",
      body: "Do you want start this Server?",
      callback: () => this.restService.startContainer(containerId).subscribe(
        {
          next: data => {
            this.actionLoading = false
            this.loadServerData()
          },
          error: err => {
          }
        })
    }

    this.showDialog(startCommand)

  }

  restartContainer(containerId: string) {
    this.actionLoading = true
    let restartCommand: ContainerCommand = {
      title: "Confirmation",
      body: "Do you want restart this Server?",
      callback: () => this.restService.restartContainer(containerId).subscribe(
        {
          next: data => {
            this.actionLoading = false
            this.loadServerData()
          },
          error: err => {
          }
        })
    }
    this.showDialog(restartCommand)
  }


  removeContainer(containerId: string) {
    let removeCommand: ContainerCommand = {
      title: "Confirmation",
      body: "Do you want to remove this Server?",
      callback: () => this.restService.deleteContainer(containerId).subscribe()
    }
    this.showDialog(removeCommand)
  }

  resetContainer(containerId: string) {
    let resetCommand: ContainerCommand = {
      title: "Reset Confirmation",
      body: "A Server Reset will stop the server, remove all .jar Files for the Server and then start the server. Plugins are not affected. ",
      callback: () => this.restService.resetContainer(containerId).subscribe()
    }
    this.showDialog(resetCommand)
  }

  showDialog(command: ContainerCommand) {
    this.dialogCommand = command
    this.confirmationDialog?.open()
  }

  dialogOnConfirmation(command: () => void) {
    command()
    this.confirmationDialog?.close()
  }


  showNewServerWizard() {
    this.wizardMedium?.open()
  }

  newServerWizardFinish() {
    let serverName = this.newServerGeneralForm.get('servername')?.value
    if (serverName != undefined) {
      this.restService.createNewServer(serverName).subscribe({
        next: data => this.loadServerData(),
        error: err => {
        }
      })
    }
  }

}

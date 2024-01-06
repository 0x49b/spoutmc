import {Component, OnDestroy, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";
import {MCServer} from "../../model/server";
import {Router, RouterLink} from "@angular/router";
import {ClrDatagridModule, ClrSelectModule} from "@clr/angular";
import {MCServerDetail} from "../../model/serverDetail";
import {LoaderComponent} from "../util/loader/loader.component";
import {FormsModule} from "@angular/forms";
import {CdsIconModule} from "@cds/angular";
import {OutlineIconsModule} from "@dimaslz/ng-heroicons";

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
    OutlineIconsModule
  ],
  templateUrl: './server.component.html',
  styleUrl: './server.component.css'
})
export class ServerComponent implements OnInit, OnDestroy {

  loading = false
  actionLoading = false
  displayedColumns: string[] = ['Names', 'State', 'Action']
  dataSource: MCServerDetail[] = []
  reloadInterval: any = null
  reload: number = 5
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

  constructor(private http: HttpClient, private router: Router) {
    this.reload = parseInt(this.readReloadTime())
  }

  ngOnInit() {
    this.loading = true
    this.loadServerData()
    this.initializeInterval()
  }

  ngOnDestroy() {
    clearInterval(this.reloadInterval)
  }

  initializeInterval() {
    window.localStorage.setItem(this.RELOAD_STORAGE_KEY, this.reload.toString())
    if (this.reload > 0) {
      this.reloadInterval = setInterval(() => {
        this.loadServerData()
      }, this.reload * 1000)
    }
  }

  setNewInterval() {
    clearInterval(this.reloadInterval)
    this.initializeInterval()
  }

  readReloadTime() {
    return window.localStorage.getItem(this.RELOAD_STORAGE_KEY) || "5";
  }


  loadServerData() {
    this.http.get<MCServerDetail[]>("http://localhost:3000/api/v1/container/withDetails").subscribe(
      data => {
        this.dataSource = []
        this.dataSource = data
        this.loading = false
      }
    )
  }

  stopContainer(containerId: string) {
    this.actionLoading = true
    this.http.get<MCServer>("http://localhost:3000/api/v1/container/stop/" + containerId).subscribe(
      data => {
        this.dataSource.forEach((server, i) => {
          if (server.Id == containerId) {
            //this.dataSource[i] = data;
            this.actionLoading = false
          }
        })

      }
    )
  }

  startContainer(containerId: string) {
    this.actionLoading = true
    this.http.get<MCServer>("http://localhost:3000/api/v1/container/start/" + containerId).subscribe(
      data => {
        this.dataSource.forEach((server, i) => {
          if (server.Id == containerId) {
            //this.dataSource[i] = data;
            this.actionLoading = false
          }
        })
      }
    )
  }

  restartContainer(containerId: string) {
    this.actionLoading = true
    this.http.get<MCServer>("http://localhost:3000/api/v1/container/restart/" + containerId).subscribe(
      data => {
        console.log(data)
        this.dataSource.forEach((server, i) => {
          if (server.Id == containerId) {
            //this.dataSource[i] = data;
            this.actionLoading = false
          }
        })
      }
    )
  }

  openNewServerDialog() {
    this.router.navigateByUrl('/server/new')
  }


  removeContainer(containerId: string) {
    this.http.delete<any>("http://localhost:3000/api/v1/container/id/" + containerId).subscribe(data => {
      console.log(data)
    })
  }
}

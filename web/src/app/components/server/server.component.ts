import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";
import {MCServer} from "../../model/server";
import {MatTableModule} from "@angular/material/table";
import {MatProgressSpinnerModule} from "@angular/material/progress-spinner";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";
import {MatIconModule} from "@angular/material/icon";
import {MatButtonModule} from "@angular/material/button";
import {RouterLink} from "@angular/router";


export interface ReloadTimes {
  value: number,
  viewValue: string
}

@Component({
  selector: 'app-server',
  standalone: true,
  imports: [
    MatTableModule,
    MatProgressSpinnerModule,
    MatInputModule,
    MatSelectModule,
    MatIconModule,
    MatButtonModule,
    RouterLink
  ],
  templateUrl: './server.component.html',
  styleUrl: './server.component.css'
})
export class ServerComponent implements OnInit {

  loading = false
  actionLoading = false
  displayedColumns: string[] = ['Names', 'State', 'Action']
  dataSource: MCServer[] = []
  reloadInterval: any = null
  reload: number = 5
  reloadTimes: ReloadTimes[] = [
    {value: 5, viewValue: '5 Seconds'},
    {value: 10, viewValue: '10 Seconds'},
    {value: 20, viewValue: '20 Seconds'},
    {value: 30, viewValue: '30 Seconds'},
    {value: 45, viewValue: '45 Seconds'},
    {value: 60, viewValue: '1 Minute'},
    {value: 0, viewValue: 'never'},

  ];

  constructor(private http: HttpClient) {
  }

  ngOnInit() {
    this.loading = true
    this.loadServerData()
    this.initializeInterval()
  }

  initializeInterval() {
    if (this.reload > 0) {
      this.reloadInterval = setInterval(() => {
        this.loadServerData()
      }, this.reload * 1000)
    }
  }

  loadServerData() {
    this.http.get<MCServer[]>("http://localhost:3000/api/v1/container").subscribe(
      data => {
        this.dataSource = data
        this.loading = false
      }
    )
  }

  setNewInterval() {
    clearInterval(this.reloadInterval)
    this.initializeInterval()
  }

  stopContainer(containerId: string) {
    this.actionLoading = true
    this.http.get<MCServer>("http://localhost:3000/api/v1/container/stop/" + containerId).subscribe(
      data => {
        this.dataSource.forEach((server, i) => {
          if (server.Id == containerId) {
            this.dataSource[i] = data;
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
            this.dataSource[i] = data;
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
            this.dataSource[i] = data;
            this.actionLoading = false
          }
        })
      }
    )
  }
}

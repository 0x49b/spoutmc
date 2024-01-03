import {Component, inject, OnInit} from '@angular/core';
import {ActivatedRoute} from "@angular/router";
import {HttpClient} from "@angular/common/http";
import {BannedPlayer} from "../../model/bannedPlayers";

@Component({
  selector: 'app-server-edit',
  standalone: true,
  imports: [],
  templateUrl: './server-edit.component.html',
  styleUrl: './server-edit.component.css'
})
export class ServerEditComponent implements OnInit {

  privateActivatedRoute = inject(ActivatedRoute)
  serverId = ""
  bannedPlayers: BannedPlayer[] = []

  constructor(private http: HttpClient) {
    this.privateActivatedRoute = inject(ActivatedRoute)
    this.serverId = this.privateActivatedRoute.snapshot.params['serverId'];
  }

  ngOnInit() {
    this.loadBannedPlayers()
  }

  loadBannedPlayers() {
    this.http.get<BannedPlayer[]>("http://localhost:3000/api/v1/container/bannedPlayers/" + this.serverId).subscribe(
      data => {
        this.bannedPlayers = data
      }
    )
  }


}

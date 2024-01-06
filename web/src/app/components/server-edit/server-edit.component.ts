import {Component, inject, OnInit} from '@angular/core';
import {ActivatedRoute} from "@angular/router";
import {HttpClient, HttpHeaders} from "@angular/common/http";
import {BannedPlayer} from "../../model/bannedPlayers";
import {FormBuilder, ReactiveFormsModule} from "@angular/forms";
import {OpPlayer} from "../../model/opPlayers";
import {MCServerDetail} from "../../model/serverDetail";

@Component({
  selector: 'app-server-edit',
  standalone: true,
  imports: [
    ReactiveFormsModule
  ],
  templateUrl: './server-edit.component.html',
  styleUrl: './server-edit.component.css'
})
export class ServerEditComponent implements OnInit {

  privateActivatedRoute = inject(ActivatedRoute)
  serverId = ""
  serverDetails: MCServerDetail | undefined = undefined
  bannedPlayersColumns: string[] = ['name', 'uuid', 'created', 'source', 'expires', 'reason', 'action']
  bannedPlayers: BannedPlayer[] = []
  opPlayersColumns: string[] = ['name', 'uuid', 'level', 'bypassesPlayerLimit', 'action']
  opPlayers: OpPlayer[] = []
  commandForm = this.formBuilder.group({command: ''})

  constructor(private http: HttpClient, private formBuilder: FormBuilder) {
    this.privateActivatedRoute = inject(ActivatedRoute)
    this.serverId = this.privateActivatedRoute.snapshot.params['serverId'];
  }

  ngOnInit() {
    this.loadServerDetails()
    this.loadBannedPlayers()
    this.loadOpPlayers()
  }

  loadServerDetails() {
    this.http.get<MCServerDetail>("http://localhost:3000/api/v1/container/id/" + this.serverId).subscribe(
      data => {
        this.serverDetails = data
        this.serverDetails.Name = data.Name.slice(1).charAt(0).toUpperCase() + data.Name.slice(2)
      }
    )
  }

  loadOpPlayers() {
    this.http.get<OpPlayer[]>("http://localhost:3000/api/v1/container/opPlayers/" + this.serverId).subscribe(
      data => {
        this.opPlayers = JSON.parse(data.toString())
      }
    )
  }

  loadBannedPlayers() {
    this.http.get<BannedPlayer[]>("http://localhost:3000/api/v1/container/bannedPlayers/" + this.serverId).subscribe(
      data => {
        this.bannedPlayers = JSON.parse(data.toString())
      }
    )
  }

  onSubmit() {

    const headers = new HttpHeaders({'Content-Type': 'application/json'});
    this.http.post<any>("http://localhost:3000/api/v1/container/command/" + this.serverId,
      {command: this.commandForm.value['command']},
      {headers}
    ).subscribe(
      () => {
        this.commandForm.reset()
        this.loadBannedPlayers()
        this.loadOpPlayers()
      }
    )
  }
}

import {Component, Inject} from '@angular/core';
import {
  MAT_DIALOG_DATA,
  MatDialogActions, MatDialogClose,
  MatDialogContent,
  MatDialogRef,
  MatDialogTitle
} from "@angular/material/dialog";
import {MatInputModule} from "@angular/material/input";
import {FormsModule} from "@angular/forms";
import {MatButtonModule} from "@angular/material/button";

export interface DialogData {
  name: string;
}

@Component({
  selector: 'app-server-new-dialog',
  standalone: true,
  imports: [
    MatInputModule,
    MatDialogTitle,
    FormsModule,
    MatButtonModule,
    MatDialogActions,
    MatDialogContent,
    MatDialogClose
  ],
  templateUrl: './server-new-dialog.component.html',
  styleUrl: './server-new-dialog.component.css'
})
export class ServerNewDialogComponent {

  constructor(public dialogRef: MatDialogRef<ServerNewDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: DialogData) {}

  onCancelClick() {
    this.dialogRef.close()
  }

}

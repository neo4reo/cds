import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {
    Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger
} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-trigger-join',
    templateUrl: './workflow.trigger.join.html',
    styleUrls: ['./workflow.trigger.join.scss']
})
export class WorkflowTriggerJoinComponent {

    @ViewChild('triggerJoinModal')
    modalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Output() triggerChange = new EventEmitter<WorkflowNodeJoinTrigger>();
    @Input() join: WorkflowNodeJoin;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNodeJoinTrigger;
    @Input() loading: boolean;

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.modalTemplate);
        this.modal = this._modalService.open(config);
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        this.triggerChange.emit(this.trigger);
    }
}

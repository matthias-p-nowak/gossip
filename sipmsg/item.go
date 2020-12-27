package sipmsg

type Item struct {
  msg *SipMsg
}

func CreateItem(msg *SipMsg) (ret *Item){
  ret=new(Item)
  ret.msg=msg
  return
}
